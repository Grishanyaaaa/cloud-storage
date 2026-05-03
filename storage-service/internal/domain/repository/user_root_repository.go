package repository

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// UserRootRepository defines persistence operations for the user → root mapping.
type UserRootRepository interface {
	// CreateTx inserts a new (user_id, root_id) row inside a transaction.
	// Returns domainerr.ErrUserRootAlreadyExists on conflict.
	CreateTx(ctx context.Context, tx Transaction, ur *entity.UserRoot) error

	// GetByUserID returns the root binding for a user.
	// Returns domainerr.ErrUserRootNotFound if not exists.
	GetByUserID(ctx context.Context, userID valueobject.UserID) (*entity.UserRoot, error)

	// ExistsByUserID returns true when a root binding exists for the user.
	ExistsByUserID(ctx context.Context, userID valueobject.UserID) (bool, error)
}
