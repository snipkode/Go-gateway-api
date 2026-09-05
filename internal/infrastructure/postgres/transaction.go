package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ctxKey int

const txKey ctxKey = iota

// contextWithTx stores a *Tx in the context so per-transaction repositories
// created via FromQuerier(ctx) bind to the running transaction.
func contextWithTx(ctx context.Context, tx *Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// FromQuerier resolves the live query interface from ctx. When a transaction
// is active it returns the transaction; otherwise it returns the pool.
func FromQuerier(ctx context.Context, pool Querier) Querier {
	if tx, ok := ctx.Value(txKey).(*Tx); ok {
		return tx
	}
	return pool
}

// IsUniqueViolation reports whether err is a PostgreSQL unique_violation
// (SQLSTATE 23505) — used, for example, to map duplicate email errors.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsNotFound reports whether err is a pgx "no rows" error.
func IsNotFound(err error) bool {
	return err != nil && errors.Is(err, pgx.ErrNoRows)
}
