package services

import (
	"context"
	"errors"
)

// errFakeTxAlreadyClosed simula o erro que um driver real retornaria ao
// tentar dar rollback numa transação já commitada.
var errFakeTxAlreadyClosed = errors.New("fakeTx: transação já foi fechada")

type fakeTx struct {
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
		return errFakeTxAlreadyClosed
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

func (f *fakeConnPool) Begin(ctx context.Context) (Tx, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}

	if f.tx.committed || f.tx.rolledBack {
		panic("fakeConnPool: Begin chamado mais de uma vez; use uma fixture nova por transação")
	}
	return f.tx, nil
}
