package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectPsql() (*pgxpool.Pool, error) {
	connStr := strings.TrimSpace(os.Getenv("DATABASE_URL"))

	if connStr == "" {
		dbUser := os.Getenv("DB_USER")
		dbPass := os.Getenv("DB_PASS")
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbName := os.Getenv("DB_NAME")

		connStr = fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser,
			dbPass,
			dbHost,
			dbPort,
			dbName,
		)
	}

	pgc, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	if err := pgc.Ping(context.Background()); err != nil {
		pgc.Close()
		return nil, err
	}

	return pgc, nil
}

// Render uses DATABASE_URL / connection string.
// Local development can still use DB_USER, DB_PASS, DB_HOST, DB_PORT, and DB_NAME.
