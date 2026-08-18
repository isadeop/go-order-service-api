package repository

import (
	"context"
	"testing"
)

func TestConnPool_BeginAbreUmaTransacaoValida(t *testing.T) {
	pool := newTestPool(t)

	connPool := NewConnPool(pool)

	tx, err := connPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin retornou erro inesperado: %v", err)
	}

	if err := tx.Rollback(context.Background()); err != nil {
		t.Errorf("Rollback retornou erro inesperado: %v", err)
	}
}
