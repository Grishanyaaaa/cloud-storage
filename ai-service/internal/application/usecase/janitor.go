package usecase

import (
	"context"
	"log/slog"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
)

// JanitorExpirePendingPlans transitions awaiting_confirmation commands whose
// expires_at has elapsed into the `expired` status.
//
// Designed to be invoked periodically (e.g. every minute) from a goroutine in
// cmd/ai-service/main.go.
type JanitorExpirePendingPlans struct {
	cmdRepo   repository.AiCommandRepository
	batchSize int
	logger    *slog.Logger
}

// NewJanitorExpirePendingPlans creates a new janitor.
func NewJanitorExpirePendingPlans(
	cmdRepo repository.AiCommandRepository,
	batchSize int,
	logger *slog.Logger,
) *JanitorExpirePendingPlans {
	if batchSize <= 0 {
		batchSize = 500
	}
	return &JanitorExpirePendingPlans{
		cmdRepo:   cmdRepo,
		batchSize: batchSize,
		logger:    logger,
	}
}

// Run executes a single tick — expires up to batchSize rows and returns the
// number of rows expired this tick.
func (j *JanitorExpirePendingPlans) Run(ctx context.Context) (int64, error) {
	n, err := j.cmdRepo.ExpirePendingBatch(ctx, j.batchSize)
	if err != nil {
		j.logger.ErrorContext(ctx, "janitor expire pending plans failed",
			slog.String("error", err.Error()))
		return 0, err
	}
	if n > 0 {
		j.logger.InfoContext(ctx, "expired pending ai commands",
			slog.Int64("count", n))
	}
	return n, nil
}
