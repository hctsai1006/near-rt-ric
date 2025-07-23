package db

import (
	"context"
	"fmt"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

// PostgresDB represents a PostgreSQL database connection
type PostgresDB struct {
	Pool *pgxpool.Pool
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.DatabaseConfig, logger *logrus.Logger) (*PostgresDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.Connect(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	// Ping the database to verify the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	logger.Info("Successfully connected to PostgreSQL database")

	return &PostgresDB{Pool: pool}, nil
}

// Close closes the database connection
func (db *PostgresDB) Close() {
	db.Pool.Close()
}
