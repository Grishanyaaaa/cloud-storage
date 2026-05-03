package repository

import "context"

// Transaction represents a database transaction abstraction.
// This interface allows the domain layer to remain independent of specific database implementations.
type Transaction interface {
	// Commit commits the transaction.
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction.
	Rollback(ctx context.Context) error
}
