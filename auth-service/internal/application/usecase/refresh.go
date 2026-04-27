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

// Refresh handles token rotation by issuing a new pair of tokens.
func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.TokenPairResponse, error) {
	now := time.Now()

	// 1. Поиск старого токена по его хешу
	tokenHash := s.tokenHasher.Hash(req.RefreshToken)
	oldToken, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if domainerr.IsNotFound(err) {
			return nil, domainerr.ErrInvalidToken
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	// 2. Проверка валидности токена (срок действия и отзыв)
	if !oldToken.IsValid(now) {
		if oldToken.RevokedAt() != nil {
			return nil, domainerr.ErrRefreshTokenRevoked
		}
		return nil, domainerr.ErrTokenExpired
	}

	// 3. Отзыв старого токена (Token Rotation)
	if err := s.tokenRepo.RevokeByID(ctx, oldToken.ID(), now); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	// 4. Получение данных пользователя
	user, err := s.userRepo.GetByID(ctx, oldToken.UserID())
	if err != nil {
		if domainerr.IsNotFound(err) {
			return nil, domainerr.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user for refresh: %w", err)
	}

	// 5. Проверка активности пользователя
	if !user.CanLogin() {
		return nil, domainerr.ErrUserInactive
	}

	// 6. Генерация новых токенов
	claims := port.TokenClaims{
		UserID: user.ID().String(),
		Email:  user.Email().String(),
	}
	accessToken, err := s.tokenManager.GenerateAccessToken(claims, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshTokenRaw, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate new refresh token: %w", err)
	}

	// 7. Сохранение нового Refresh токена
	newRefreshTokenHash := s.tokenHasher.Hash(newRefreshTokenRaw)
	expiresAt := now.Add(s.tokenManager.RefreshTokenTTL())

	newRefreshToken := entity.NewRefreshToken(
		valueobject.NewRefreshTokenID(),
		user.ID(),
		newRefreshTokenHash,
		expiresAt,
		req.IPAddress,
		req.UserAgent,
		now,
	)

	if err := s.tokenRepo.Save(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("save new refresh token: %w", err)
	}

	// 8. Лог аудита
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		user.ID(),
		entity.ActionRefresh,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	_ = s.auditRepo.Save(ctx, auditLog)

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenRaw,
	}, nil
}
