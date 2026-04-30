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
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/database"
)

// Refresh handles token rotation by issuing a new pair of tokens.
func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.TokenPairResponse, error) {
	now := time.Now()

	// 1. Поиск токена по хешу
	tokenHash := s.tokenHasher.Hash(req.RefreshToken)

	// 2. Получение данных пользователя (вне транзакции для оптимизации)
	var oldToken *entity.RefreshToken
	var wasRevokedAt *time.Time
	var user *entity.User

	// 3. Атомарный отзыв старого токена и сохранение нового в транзакции
	var accessToken string
	var newRefreshTokenRaw string

	err := database.WithTransaction(ctx, s.pool, func(ctx context.Context, tx repository.Transaction) error {
		// Revoke old token atomically
		var err error
		oldToken, wasRevokedAt, err = s.tokenRepo.RevokeByHashTx(ctx, tx, tokenHash, now)
		if err != nil {
			if domainerr.IsNotFound(err) {
				return domainerr.ErrInvalidToken
			}
			return fmt.Errorf("revoke and find refresh token: %w", err)
		}

		// Check validity (before we revoked it just now)
		if wasRevokedAt != nil {
			return domainerr.ErrRefreshTokenRevoked
		}
		if !oldToken.ExpiresAt().After(now) {
			return domainerr.ErrTokenExpired
		}

		// Get user data
		user, err = s.userRepo.GetByID(ctx, oldToken.UserID())
		if err != nil {
			if domainerr.IsNotFound(err) {
				return domainerr.ErrUserNotFound
			}
			return fmt.Errorf("get user for refresh: %w", err)
		}

		// Check user is active
		if !user.CanLogin() {
			return domainerr.ErrUserInactive
		}

		// Generate new tokens
		claims := port.TokenClaims{
			UserID: user.ID().String(),
			Email:  user.Email().String(),
		}
		accessToken, err = s.tokenManager.GenerateAccessToken(claims, now)
		if err != nil {
			return fmt.Errorf("generate access token: %w", err)
		}

		newRefreshTokenRaw, err = s.tokenManager.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("generate new refresh token: %w", err)
		}

		// Save new refresh token
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

		if err := s.tokenRepo.SaveTx(ctx, tx, newRefreshToken); err != nil {
			return fmt.Errorf("save new refresh token: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 4. Лог аудита (вне транзакции, так как его сбой не критичен)
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		user.ID(),
		entity.ActionRefresh,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.WarnContext(ctx, "failed to save audit log", "error", err, "action", "refresh", "user_id", user.ID().String())
	}

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenRaw,
	}, nil
}
