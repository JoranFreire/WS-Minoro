package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a single shared connection pool used by every repository.
func Connect(databaseURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		panic("failed to connect to postgres: " + err.Error())
	}
	return pool
}
