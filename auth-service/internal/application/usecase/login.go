package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// Login handles user authentication and issues tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPairResponse, error) {
	// 1. Валидация email
	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return nil, domainerr.ErrInvalidCredentials // Маскируем ошибки валидации для безопасности
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

	// 5. Обновление времени последнего входа
	now := time.Now()
	user.UpdateLastLogin(now)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user last login: %w", err)
	}

	// 6. Генерация Access Token
	claims := port.TokenClaims{
		UserID: user.ID().String(),
		Email:  user.Email().String(),
	}
	accessToken, err := s.tokenManager.GenerateAccessToken(claims, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 7. Генерация Refresh Token
	refreshTokenRaw, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 8. Сохранение Refresh Token в БД
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

	if err := s.tokenRepo.Save(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	// 9. Лог аудита
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		user.ID(),
		entity.ActionLogin,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	_ = s.auditRepo.Save(ctx, auditLog)

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
	}, nil
}
