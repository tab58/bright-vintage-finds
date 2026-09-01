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
}

func NewClient(config ClientConfig) (*Client, error) {
	dbConn, err := sql.Open("pgx", config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, dbConn)
	client := generated.NewClient(generated.Driver(drv))

	return &Client{
		db: client,
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
