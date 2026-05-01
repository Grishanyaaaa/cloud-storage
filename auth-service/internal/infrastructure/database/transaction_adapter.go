package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
)

// Compile-time check: pgxTransactionAdapter implements repository.Transaction
var _ repository.Transaction = (*pgxTransactionAdapter)(nil)

// pgxTransactionAdapter adapts pgx.Tx to the domain Transaction interface.
type pgxTransactionAdapter struct {
	tx pgx.Tx
}

// newTransactionAdapter wraps a pgx.Tx into a domain Transaction interface.
func newTransactionAdapter(tx pgx.Tx) repository.Transaction {
	return &pgxTransactionAdapter{tx: tx}
}

// Commit commits the transaction.
func (a *pgxTransactionAdapter) Commit(ctx context.Context) error {
	return a.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (a *pgxTransactionAdapter) Rollback(ctx context.Context) error {
	return a.tx.Rollback(ctx)
}

// unwrapTx extracts the underlying pgx.Tx from a Transaction interface.
// This is used internally by repository implementations.
// Returns an error if the transaction is not a pgxTransactionAdapter, which indicates
// a programming error (wrong transaction type passed to repository method).
func unwrapTx(tx repository.Transaction) (pgx.Tx, error) {
	if adapter, ok := tx.(*pgxTransactionAdapter); ok {
		return adapter.tx, nil
	}
	return nil, fmt.Errorf("transaction is not a pgxTransactionAdapter")
}
