package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// CleanupExpiredTokens removes expired refresh tokens from the database.
// Should be called periodically (e.g., daily via cron job).
// Returns the number of deleted tokens.
func (s *AuthService) CleanupExpiredTokens(ctx context.Context, log *slog.Logger) (int64, error) {
	now := time.Now()

	deleted, err := s.tokenRepo.DeleteExpired(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired tokens: %w", err)
	}

	if deleted > 0 {
		log.Info("cleaned up expired refresh tokens", slog.Int64("deleted", deleted))
	}

	return deleted, nil
}
