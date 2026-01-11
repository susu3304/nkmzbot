package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type DB struct {
	pool        *pgxpool.Pool
	databaseURL string
}

func New(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		pool:        pool,
		databaseURL: databaseURL,
	}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

// Exec executes a query without returning any rows.
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// Query executes a query that returns rows.
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// RunMigrations runs database migrations using pressly/goose
func (db *DB) RunMigrations(ctx context.Context) error {
	// Goose requires a *sql.DB.
	// We open a temporary connection using pgx stdlib adapter for migration purposes.
	// Since we already have the connection string, we can use sql.Open with "pgx".
	// Make sure "github.com/jackc/pgx/v5/stdlib" is imported for side-effects if we used "pgx" driver name,
	// but here we can't rely on driver registration without side-effect import.
	// Actually stdlib.OpenDB works with a config, but sql.Open works with a string.
	// To keep it simple and reuse the URL string:

	connConfig, err := pgx.ParseConfig(db.databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse config for migrations: %w", err)
	}

	// Open a standard library database connection
	sqldb := stdlib.OpenDB(*connConfig)
	defer sqldb.Close()

	if err := sqldb.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping standard lib db: %w", err)
	}

	// Set migration directory
	migrationsDir := "./migrations"

	// Configure goose
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations
	if err := goose.UpContext(ctx, sqldb, migrationsDir); err != nil {
		return fmt.Errorf("failed to run goose up: %w", err)
	}

	return nil
}
