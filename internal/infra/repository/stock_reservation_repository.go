package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/txport"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertStockReservationQuery = `
		INSERT INTO stock_reservations (saga_id, product_id, quantity, status)
		VALUES ($1, $2, $3, $4)
	`

	findStockReservationForUpdateQuery = `
		SELECT saga_id, product_id, quantity, status, created_at, updated_at
		FROM stock_reservations
		WHERE saga_id = $1 AND product_id = $2
		FOR UPDATE
	`

	updateStockReservationStatusQuery = `
		UPDATE stock_reservations
		SET status = $3, updated_at = now()
		WHERE saga_id = $1 AND product_id = $2
	`

	findStaleReservedQuery = `
		SELECT saga_id, product_id, quantity, status, created_at, updated_at
		FROM stock_reservations
		WHERE status = $1 AND created_at < $2
		ORDER BY created_at
	`
)

type StockReservationRepository struct {
	pool *pgxpool.Pool
}

func NewStockReservationRepository(pool *pgxpool.Pool) *StockReservationRepository {
	return &StockReservationRepository{pool: pool}
}

// FindForUpdate busca e trava a reserva de uma saga para um produto
func (repo *StockReservationRepository) FindForUpdate(
	ctx context.Context,
	tx txport.Tx,
	sagaID uuid.UUID,
	productID uuid.UUID,
) (domain.StockReservation, error) {

	var reservation domain.StockReservation

	pTx, err := pgxTx(tx)
	if err != nil {
		return domain.StockReservation{}, err
	}

	err = pTx.QueryRow(ctx, findStockReservationForUpdateQuery, sagaID, productID).Scan(
		&reservation.SagaID,
		&reservation.ProductID,
		&reservation.Quantity,
		&reservation.Status,
		&reservation.CreatedAt,
		&reservation.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StockReservation{}, custom_errors.ErrStockReservationNotFound
	}

	if err != nil {
		return domain.StockReservation{}, fmt.Errorf("find stock reservation for update: %w", err)
	}

	return reservation, nil
}

// Create registra que uma reserva foi aplicada
func (repo *StockReservationRepository) Create(
	ctx context.Context,
	tx txport.Tx,
	reservation domain.StockReservation,
) error {

	pTx, err := pgxTx(tx)
	if err != nil {
		return err
	}

	_, err = pTx.Exec(
		ctx,
		insertStockReservationQuery,
		reservation.SagaID,
		reservation.ProductID,
		reservation.Quantity,
		reservation.Status,
	)

	if err != nil {
		return fmt.Errorf("create stock reservation: %w", err)
	}

	return nil
}

func (repo *StockReservationRepository) UpdateStatus(
	ctx context.Context,
	tx txport.Tx,
	sagaID uuid.UUID,
	productID uuid.UUID,
	status domain.StockReservationStatus,
) error {

	pTx, err := pgxTx(tx)
	if err != nil {
		return err
	}

	commandTag, err := pTx.Exec(ctx, updateStockReservationStatusQuery, sagaID, productID, status)
	if err != nil {
		return fmt.Errorf("update stock reservation status: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return custom_errors.ErrStockReservationNotFound
	}

	return nil
}

// FindStaleReserved lista reservas ainda com status RESERVED cuja created_at
// é anterior a olderThan
func (repo *StockReservationRepository) FindStaleReserved(
	ctx context.Context,
	olderThan time.Time,
) ([]domain.StockReservation, error) {

	rows, err := repo.pool.Query(ctx, findStaleReservedQuery, domain.StockReservationStatusReserved, olderThan)
	if err != nil {
		return nil, fmt.Errorf("find stale reserved stock reservations: %w", err)
	}
	defer rows.Close()

	reservations := make([]domain.StockReservation, 0)

	for rows.Next() {
		var reservation domain.StockReservation
		err := rows.Scan(
			&reservation.SagaID,
			&reservation.ProductID,
			&reservation.Quantity,
			&reservation.Status,
			&reservation.CreatedAt,
			&reservation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stock reservation: %w", err)
		}
		reservations = append(reservations, reservation)
	}

	return reservations, rows.Err()
}
