package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
)

// Compile-time check: pgxTransactionAdapter implements repository.Transaction
var _ repository.Transaction = (*pgxTransactionAdapter)(nil)

// pgxTransactionAdapter adapts pgx.Tx to the domain Transaction interface.
type pgxTransactionAdapter struct {
	tx pgx.Tx
}

func newTransactionAdapter(tx pgx.Tx) repository.Transaction {
	return &pgxTransactionAdapter{tx: tx}
}

func (a *pgxTransactionAdapter) Commit(ctx context.Context) error {
	return a.tx.Commit(ctx)
}

func (a *pgxTransactionAdapter) Rollback(ctx context.Context) error {
	return a.tx.Rollback(ctx)
}

// unwrapTx extracts the underlying pgx.Tx from a domain Transaction.
// Used by repository implementations.
func unwrapTx(tx repository.Transaction) (pgx.Tx, error) {
	if adapter, ok := tx.(*pgxTransactionAdapter); ok {
		return adapter.tx, nil
	}
	return nil, fmt.Errorf("transaction is not a pgxTransactionAdapter")
}
