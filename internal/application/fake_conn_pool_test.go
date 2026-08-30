package application

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
	tx       *fakeTx   // última transação aberta
	txs      []*fakeTx // histórico completo
	beginErr error
}

func newFakeConnPool() *fakeConnPool {
	pool := &fakeConnPool{}
	pool.tx = &fakeTx{}
	pool.txs = []*fakeTx{pool.tx}
	return pool
}

func (f *fakeConnPool) Begin(ctx context.Context) (Tx, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}

	if !f.tx.committed && !f.tx.rolledBack {
		return f.tx, nil
	}

	tx := &fakeTx{}
	f.tx = tx
	f.txs = append(f.txs, tx)
	return tx, nil
}
