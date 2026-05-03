package usecase

import (
	"context"
	"time"
)

const janitorBatchLimit = 500

// JanitorExpirePendingUploads marks pending blobs whose pre-signed URL is past TTL as failed.
// Designed to be called periodically by a goroutine in main.go.
func (s *StorageService) JanitorExpirePendingUploads(ctx context.Context) (int64, error) {
	return s.blobRepo.FailExpiredPending(ctx, time.Now(), janitorBatchLimit)
}

// JanitorExpireShares marks shares whose expires_at ≤ now as revoked.
func (s *StorageService) JanitorExpireShares(ctx context.Context) (int64, error) {
	return s.shareRepo.ExpireDue(ctx, time.Now(), janitorBatchLimit)
}
