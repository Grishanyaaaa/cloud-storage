package repository

import "context"

// TransactionManager manages database transactions.
// This interface allows the application layer to execute transactional operations
// without depending on specific database implementations.
type TransactionManager interface {
	// WithTransaction executes a function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	WithTransaction(ctx context.Context, fn func(ctx context.Context, tx Transaction) error) error
}
