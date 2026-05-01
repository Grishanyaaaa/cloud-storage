package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
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

	// 3. Prepare hash to compare (dummy hash if user not found to prevent timing attack)
	// This ensures bcrypt.Compare always runs, making response time consistent
	var hashToCompare string
	if err == nil {
		hashToCompare = user.PasswordHash()
	} else {
		// Pre-generated bcrypt hash with cost 10 (same as real passwords)
		// This is a hash of "dummy-password-for-timing-attack-prevention"
		hashToCompare = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	}

	// 4. Always perform password comparison to prevent timing attack
	matches, compareErr := s.hasher.Compare(hashToCompare, req.Password)
	if compareErr != nil {
		return nil, fmt.Errorf("compare passwords: %w", compareErr)
	}

	// 5. Now check if user lookup failed or password doesn't match
	if err != nil {
		if domainerr.IsNotFound(err) {
			return nil, domainerr.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if !matches {
		return nil, domainerr.ErrInvalidCredentials
	}

	// 6. Проверка активности пользователя
	if !user.CanLogin() {
		return nil, domainerr.ErrUserInactive
	}

	now := time.Now()

	// 7. Генерация Access Token
	claims := port.TokenClaims{
		UserID: user.ID().String(),
		Email:  user.Email().String(),
	}
	accessToken, err := s.tokenManager.GenerateAccessToken(claims, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 8. Генерация Refresh Token
	refreshTokenRaw, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 9. Атомарное обновление last_login и сохранение refresh token в транзакции
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

	err = s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
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

	// 10. Лог аудита (вне транзакции, так как его сбой не критичен)
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		user.ID(),
		entity.ActionLogin,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.WarnContext(ctx, "failed to save audit log", "error", err, "action", "login", "user_id", user.ID().String())
	}

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
	}, nil
}
