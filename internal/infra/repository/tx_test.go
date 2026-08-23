package repository

import (
	"context"
	"testing"
)

type notAPgxTx struct{}

func (notAPgxTx) Commit(ctx context.Context) error   { return nil }
func (notAPgxTx) Rollback(ctx context.Context) error { return nil }

func TestPgxTx_TipoInesperadoRetornaErro(t *testing.T) {
	_, err := pgxTx(notAPgxTx{})
	if err == nil {
		t.Fatal("esperava erro ao converter um txport.Tx que não é um pgx.Tx real")
	}
}
