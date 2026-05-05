package repository

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// AiCommandRepository defines persistence operations for AiCommand entities.
// Implemented in infrastructure layer. Returns sentinels on known failures.
type AiCommandRepository interface {
	// Create persists a new ai_command.
	Create(ctx context.Context, cmd *entity.AiCommand) error

	// GetByID retrieves a command by ID.
	// Returns domainerr.ErrCommandNotFound if not exists.
	GetByID(ctx context.Context, id valueobject.CommandID) (*entity.AiCommand, error)

	// GetByIDTx retrieves a command by ID within a transaction (typically with FOR UPDATE).
	// Returns domainerr.ErrCommandNotFound if not exists.
	GetByIDTx(ctx context.Context, tx Transaction, id valueobject.CommandID) (*entity.AiCommand, error)

	// UpdateTx persists changes to an existing ai_command within a transaction.
	UpdateTx(ctx context.Context, tx Transaction, cmd *entity.AiCommand) error

	// ExpirePendingBatch transitions up to `limit` awaiting_confirmation commands
	// whose expires_at < `now` into `expired` status. Returns the number of rows updated.
	// Used by janitor.
	ExpirePendingBatch(ctx context.Context, limit int) (int64, error)
}
