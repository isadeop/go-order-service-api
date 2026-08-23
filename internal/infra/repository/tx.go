package repository

import (
	"fmt"

	"github.com/isadeop/go-order-service-api/internal/txport"
	"github.com/jackc/pgx/v5"
)

// pgxTx recupera o pgx.Tx concreto por trás da abstração txport.Tx.
// aplicação só conhece o ciclo de vida (Commit/Rollback) definido em txport.Tx.
func pgxTx(tx txport.Tx) (pgx.Tx, error) {

	concrete, ok := tx.(pgx.Tx)

	if !ok {
		return nil, fmt.Errorf("repository: tipo de transação inesperado: %T", tx)
	}

	return concrete, nil
}
