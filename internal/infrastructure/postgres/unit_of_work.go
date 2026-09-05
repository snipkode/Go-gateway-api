package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tx is the shared transaction holder that implements multiple repository
// interfaces so a Unit of Work can expose the same transaction to several
// repositories (user update + audit insert) atomically.
type Tx struct {
	q pgx.Tx
}

func (t *Tx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.q.Query(ctx, sql, args...)
}

func (t *Tx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.q.QueryRow(ctx, sql, args...)
}

func (t *Tx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.q.Exec(ctx, sql, args...)
}

// Querier abstracts pgxpool.Pool and pgx.Tx so repositories work in both
// normal and transactional contexts.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

var _ Querier = (*pgxpool.Pool)(nil)
var _ Querier = (*Tx)(nil)

// UnitOfWork represents a single DB transaction that can be shared across
// repositories and an in-transaction audit logger.
type UnitOfWork interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}

// PostgresUnitOfWork runs fn inside a single transaction; fn receives a
// context that carries the Tx. All repositories in this codebase resolve the
// transaction from the context (transaction.go / FromQuerier), so mutations
// issued inside fn — including audit inserts — are committed or rolled back
// together.
type PostgresUnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *PostgresUnitOfWork {
	return &PostgresUnitOfWork{pool: pool}
}

// Within executes a transaction.
func (u *PostgresUnitOfWork) Within(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txCtx := contextWithTx(ctx, &Tx{q: tx})
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
