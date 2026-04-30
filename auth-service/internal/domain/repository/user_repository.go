package repository

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// UserRepository defines the interface for user persistence operations.
// Implemented in infrastructure layer. Returns  sentinels on known failures.
type UserRepository interface {
	// Create persists a new user.
	// Returns domainerr.ErrUserAlreadyExists if email is taken.
	Create(ctx context.Context, user *entity.User) error

	// GetByID retrieves a user by their ID.
	// Returns domainerr.ErrUserNotFound if not exists.
	GetByID(ctx context.Context, id valueobject.UserID) (*entity.User, error)

	// GetByEmail retrieves a user by email.
	// Returns domainerr.ErrUserNotFound if not exists.
	GetByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error)

	// Update updates an existing user.
	// Returns domainerr.ErrUserNotFound if not exists.
	Update(ctx context.Context, user *entity.User) error

	// UpdateTx updates an existing user within a transaction.
	UpdateTx(ctx context.Context, tx Transaction, user *entity.User) error

	// ExistsByEmail checks if a user with the given email exists.
	ExistsByEmail(ctx context.Context, email valueobject.Email) (bool, error)
}
