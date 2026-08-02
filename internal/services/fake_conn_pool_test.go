package services

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type fakeTx struct {
	pgx.Tx

	committed  bool
	rolledBack bool
	commitErr  error
}

func (f *fakeTx) Commit(ctx context.Context) error {
	f.committed = true
	return f.commitErr
}

func (f *fakeTx) Rollback(ctx context.Context) error {
	if f.committed {
		return pgx.ErrTxClosed
	}
	f.rolledBack = true
	return nil
}

type fakeConnPool struct {
	tx       *fakeTx
	beginErr error
}

func newFakeConnPool() *fakeConnPool {
	return &fakeConnPool{tx: &fakeTx{}}
}

func (f *fakeConnPool) Begin(ctx context.Context) (pgx.Tx, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}

	if f.tx.committed || f.tx.rolledBack {
		panic("fakeConnPool: Begin chamado mais de uma vez; use uma fixture nova por transação")
	}
	return f.tx, nil
}
