package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func GetQueryByName[R any](ctx context.Context, q Querier, query string, args ...any) (*R, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rVal, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[R])
	if err != nil {
		return nil, err
	}

	return &rVal, nil
}

func GetSliceQueryByName[R any](ctx context.Context, q Querier, query string, args ...any) ([]R, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rVal, err := pgx.CollectRows(rows, pgx.RowToStructByName[R])
	if err != nil {
		return nil, err
	}

	return rVal, nil
}

func GetSliceQueryByType[R any](ctx context.Context, q Querier, query string, args ...any) ([]R, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rVal, err := pgx.CollectRows(rows, pgx.RowTo[R])
	if err != nil {
		return nil, err
	}

	return rVal, nil
}
