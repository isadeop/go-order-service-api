package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/isadeop/go-order-service-api/internal/infra/config"
)

// newStockTestPool conecta ao banco do stock-service (stock_db)
func newStockTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	_ = godotenv.Load()

	database := config.DatabaseConfig{
		Host:     getEnvOrDefault("STOCK_POSTGRES_HOST", "localhost"),
		Port:     getEnvOrDefault("STOCK_POSTGRES_PORT", "5433"),
		User:     getEnvOrDefault("STOCK_POSTGRES_USER", "adm"),
		Password: getEnvOrDefault("STOCK_POSTGRES_PASSWORD", "adm"),
		Name:     getEnvOrDefault("STOCK_POSTGRES_DB", "stock_db"),
		SSLMode:  getEnvOrDefault("STOCK_POSTGRES_SSLMODE", "disable"),
	}

	pool, err := pgxpool.New(context.Background(), database.URL())
	if err != nil {
		t.Skipf("não foi possível conectar ao postgres de teste (stock_db): %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("postgres de teste indisponível (stock_db): %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
