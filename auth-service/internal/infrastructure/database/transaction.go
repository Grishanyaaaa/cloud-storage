package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
)

// TxFunc is a function that executes within a transaction.
type TxFunc func(ctx context.Context, tx repository.Transaction) error

// WithTransaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	pgxTx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Wrap pgx.Tx in domain Transaction interface
	tx := newTransactionAdapter(pgxTx)

	// Ensure rollback on panic or error
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx) // Rollback on panic; error not critical as we're re-panicking
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback transaction: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
