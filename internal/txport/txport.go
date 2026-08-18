package txport

import "context"

// Package txport define o contrato mínimo/interface de transação compartilhada entre service e repo
// Tx representa uma transação em andamento. Só expõe Commit/Rollback.
// Executação de SQL será feita apenas no repo
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// ConnPool abre transações
type ConnPool interface {
	Begin(ctx context.Context) (Tx, error)
}
