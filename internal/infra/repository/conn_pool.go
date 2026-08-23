package repository

import (
	"context"

	"github.com/isadeop/go-order-service-api/internal/txport"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnPool adapta *pgxpool.Pool à porta txport.ConnPool
// é o único lugar em que a camada de aplicação passa a abrir uma transação real
type ConnPool struct {
	pool *pgxpool.Pool
}

// NewConnPool cria o adapter a partir de um pool de conexões já aberto.
func NewConnPool(pool *pgxpool.Pool) *ConnPool {
	return &ConnPool{pool: pool}
}

func (c *ConnPool) Begin(ctx context.Context) (txport.Tx, error) {
	return c.pool.Begin(ctx)
}
