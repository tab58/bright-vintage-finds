package db

import (
	"context"
	"database/sql"
	"fmt"
	"main-api/db/generated"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type ClientConfig struct {
	ConnectionString string
}

type Client struct {
	db *generated.Client
	// raw is the underlying pool; exposed for harness use (test truncation,
	// liveness checks), not for feature queries.
	raw *sql.DB
}

func NewClient(config ClientConfig) (*Client, error) {
	dbConn, err := sql.Open("pgx", config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, dbConn)
	client := generated.NewClient(generated.Driver(drv))

	return &Client{
		db:  client,
		raw: dbConn,
	}, nil
}

// NewClientFromDB wraps an existing generated.Client. This is useful for tests
// that create the Ent client via enttest.Open (which auto-runs migrations).
func NewClientFromDB(client *generated.Client) *Client {
	return &Client{db: client}
}

func (repo *Client) GetDBFromContext(ctx context.Context) *generated.Client {
	return repo.db
}

// Raw exposes the underlying database/sql pool.
func (repo *Client) Raw() *sql.DB {
	return repo.raw
}
