package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// Logout handles user logout by revoking the refresh token.
func (s *AuthService) Logout(ctx context.Context, req dto.LogoutRequest) error {
	now := time.Now()

	// 1. Поиск токена по хешу
	tokenHash := s.tokenHasher.Hash(req.RefreshToken)
	token, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if domainerr.IsNotFound(err) {
			// Если токен не найден, считаем, что выход уже выполнен (идемпотентность)
			return nil
		}
		return fmt.Errorf("find refresh token for logout: %w", err)
	}

	// 2. Отзыв токена
	if err := s.tokenRepo.RevokeByID(ctx, token.ID(), now); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	// 3. Лог аудита
	auditLog := entity.NewAuditLog(
		valueobject.NewAuditLogID(),
		token.UserID(),
		entity.ActionLogout,
		req.IPAddress,
		req.UserAgent,
		now,
	)
	// Игнорируем ошибку сохранения audit log - логирование будет в presentation layer
	_ = s.auditRepo.Save(ctx, auditLog)

	return nil
}
