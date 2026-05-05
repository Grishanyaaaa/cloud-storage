package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
)

// TxFunc is a function that executes within a transaction.
type TxFunc func(ctx context.Context, tx repository.Transaction) error

// WithTransaction executes a function within a database transaction.
//
//	* If `fn` succeeds → commit. Errors from commit are returned.
//	* If `fn` fails    → rollback. The original error is preserved as the
//	  primary one. If rollback ALSO fails, both errors are joined via
//	  errors.Join so that errors.Is/errors.As can match either.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	pgxTx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	tx := newTransactionAdapter(pgxTx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return errors.Join(err, fmt.Errorf("rollback transaction: %w", rbErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
