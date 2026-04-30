package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/database"
)

// Login handles user authentication and issues tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPairResponse, error) {
	// 1. Поиск пользователя по email (без предварительной валидации для защиты от timing attack)
	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		// Валидация не прошла, но возвращаем generic ошибку без раскрытия деталей
		return nil, domainerr.ErrInvalidCredentials
	}

	// 2. Поиск пользователя
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if domainerr.IsNotFound(err) {
			return nil, domainerr.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	// 3. Проверка активности пользователя
	if !user.CanLogin() {
		return nil, domainerr.ErrUserInactive
	}

	// 4. Проверка пароля
	matches, err := s.hasher.Compare(user.PasswordHash(), req.Password)
	if err != nil {
		return nil, fmt.Errorf("compare passwords: %w", err)
	}
	if !matches {
		return nil, domainerr.ErrInvalidCredentials
	}

	now := time.Now()

	// 5. Генерация Access Token
	claims := port.TokenClaims{
		UserID: user.ID().String(),
		Email:  user.Email().String(),
	}
	accessToken, err := s.tokenManager.GenerateAccessToken(claims, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 6. Генерация Refresh Token
	refreshTokenRaw, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 7. Атомарное обновление last_login и сохранение refresh token в транзакции
	refreshTokenHash := s.tokenHasher.Hash(refreshTokenRaw)
	expiresAt := now.Add(s.tokenManager.RefreshTokenTTL())

	refreshToken := entity.NewRefreshToken(
		valueobject.NewRefreshTokenID(),
		user.ID(),
		refreshTokenHash,
		expiresAt,
		req.IPAddress,
		req.UserAgent,
		now,
	)

	err = database.WithTransaction(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Update user last login
		user.UpdateLastLogin(now)
		if err := s.userRepo.UpdateTx(ctx, tx, user); err != nil {
			return fmt.Errorf("update user last login: %w", err)
		}

		// Save refresh token
		if err := s.tokenRepo.SaveTx(ctx, tx, refreshToken); err != nil {
			return fmt.Errorf("save refresh token: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 8. Лог аудита (вне транзакции, так как его сбой не критичен)
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		user.ID(),
		entity.ActionLogin,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.Error("failed to save audit log", "error", err, "user_id", user.ID().String(), "action", entity.ActionLogin)
	}

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
	}, nil
}
