// Package postgres implements the repository ports with pgx on Neon / PostgreSQL.
// Every query that touches user data is scoped by user_id (D6, 6.4).
// DesignDocument/7_BackendDesign.md 7.2.3 / 8_DBDesign.md
package postgres

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
)

// DB wraps a connection pool and implements repository.TxManager.
type DB struct {
	Pool *pgxpool.Pool
}

// Open connects to databaseURL.
func Open(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// Close closes the pool.
func (d *DB) Close() { d.Pool.Close() }

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// q returns the transaction in ctx, or the pool.
func (d *DB) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return d.Pool
}

// WithinTx runs fn in a transaction. Nested calls reuse the outer transaction.
func (d *DB) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// mapErr converts pgx errors to repository sentinels.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return repository.ErrConflict
		case "23503":
			return repository.ErrNotFound
		}
	}
	return err
}

func affected(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// keyset is DESC keyset pagination over a list of columns with SQL casts.
type keyset struct {
	cols  []string
	casts []string
}

var (
	byCreated = keyset{cols: []string{"created_at", "id"}, casts: []string{"timestamptz", "uuid"}}
	byUpdated = keyset{cols: []string{"updated_at", "id"}, casts: []string{"timestamptz", "uuid"}}
)

func (k keyset) prefixed(alias string) keyset {
	cols := make([]string, len(k.cols))
	for i, c := range k.cols {
		cols[i] = alias + "." + c
	}
	return keyset{cols: cols, casts: k.casts}
}

func (k keyset) orderBy() string {
	parts := make([]string, len(k.cols))
	for i, c := range k.cols {
		parts[i] = c + " DESC"
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

// where appends the cursor condition and returns the SQL fragment and args.
func (k keyset) where(cursor string, args []any) (string, []any, error) {
	if cursor == "" {
		return "", args, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", args, errInvalidCursor
	}
	vals := strings.Split(string(raw), "|")
	if len(vals) != len(k.cols) {
		return "", args, errInvalidCursor
	}
	ph := make([]string, len(vals))
	for i, v := range vals {
		args = append(args, v)
		ph[i] = fmt.Sprintf("$%d::%s", len(args), k.casts[i])
	}
	return " AND (" + strings.Join(k.cols, ", ") + ") < (" + strings.Join(ph, ", ") + ")", args, nil
}

var errInvalidCursor = repository.ErrInvalidCursor

func encodeCursor(vals ...string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Join(vals, "|")))
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// page trims a limit+1 result to limit and computes the next cursor.
func page[T any](items []T, limit int, cursorOf func(T) string) repository.Page[T] {
	if len(items) > limit {
		next := cursorOf(items[limit-1])
		return repository.Page[T]{Items: items[:limit], NextCursor: &next}
	}
	if items == nil {
		items = []T{}
	}
	return repository.Page[T]{Items: items}
}

func statusStrings(st []vo.SessionStatus) []string {
	out := make([]string, len(st))
	for i, s := range st {
		out[i] = string(s)
	}
	return out
}

func nonNil(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}
