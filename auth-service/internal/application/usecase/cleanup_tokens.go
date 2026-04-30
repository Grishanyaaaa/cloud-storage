package usecase

import (
	"context"
	"fmt"
	"time"
)

// CleanupExpiredTokens removes expired refresh tokens from the database.
// Should be called periodically (e.g., daily via cron job).
// Returns the number of deleted tokens.
func (s *AuthService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	now := time.Now()

	deleted, err := s.tokenRepo.DeleteExpired(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired tokens: %w", err)
	}

	return deleted, nil
}
