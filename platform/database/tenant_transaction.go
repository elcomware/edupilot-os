package database

import (
	"context"
	"database/sql"
)

type transactionKey struct{}

type TenantTransaction struct {
	School string
	Pool   Pool
	Tx     *sql.Tx
}

// WithTenantTransaction lends a transaction to synchronous public module
// queries. The owner commits or rolls back after every query returns.
func WithTenantTransaction(ctx context.Context, scope TenantTransaction) context.Context {
	return context.WithValue(ctx, transactionKey{}, scope)
}

func TransactionFromContext(ctx context.Context) (TenantTransaction, bool) {
	scope, ok := ctx.Value(transactionKey{}).(TenantTransaction)
	return scope, ok
}
