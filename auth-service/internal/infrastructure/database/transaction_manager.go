package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
)

// Compile-time check: PostgresTransactionManager implements repository.TransactionManager
var _ repository.TransactionManager = (*PostgresTransactionManager)(nil)

// PostgresTransactionManager implements TransactionManager for PostgreSQL.
type PostgresTransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager creates a new PostgresTransactionManager.
func NewTransactionManager(pool *pgxpool.Pool) *PostgresTransactionManager {
	return &PostgresTransactionManager{pool: pool}
}

// WithTransaction executes a function within a database transaction.
func (m *PostgresTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx repository.Transaction) error) error {
	return WithTransaction(ctx, m.pool, fn)
}
