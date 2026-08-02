package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertOrderQuery = `
		INSERT INTO orders
		(
			client_id,
			status,
			total
		)
		VALUES
		(
			$1,
			$2,
			$3
		)
		RETURNING
			id,
			client_id,
			status,
			total,
			created_at
	`

	findAllOrdersQuery = `
		SELECT
			id,
			client_id,
			status,
			total,
			created_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1
		OFFSET $2
	`

	findOrderByIDQuery = `
		SELECT
			id,
			client_id,
			status,
			total,
			created_at
		FROM orders
		WHERE id = $1
	`

	findOrderByIDForUpdateQuery = `
		SELECT
			id,
			client_id,
			status,
			total,
			created_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`

	updateOrderTotalQuery = `
		UPDATE orders
		SET total = $2
		WHERE id = $1
	`

	updateOrderStatusQuery = `
		UPDATE orders
		SET status = $2
		WHERE id = $1
		RETURNING
			id,
			client_id,
			status,
			total,
			created_at
	`
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool: pool,
	}
}

func (repo *OrderRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	order model.Order,
) (model.Order, error) {

	err := tx.QueryRow(
		ctx,
		insertOrderQuery,
		order.ClientID,
		order.Status,
		order.Total,
	).Scan(
		&order.ID,
		&order.ClientID,
		&order.Status,
		&order.Total,
		&order.CreatedAt,
	)

	if err != nil {
		return model.Order{}, fmt.Errorf("create order: %w", err)
	}

	return order, nil
}

func (repo *OrderRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (model.Order, error) {

	var order model.Order

	err := repo.pool.QueryRow(
		ctx,
		findOrderByIDQuery,
		id,
	).Scan(
		&order.ID,
		&order.ClientID,
		&order.Status,
		&order.Total,
		&order.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, custom_errors.ErrOrderNotFound
	}

	if err != nil {
		return model.Order{}, fmt.Errorf("find order by id: %w", err)
	}

	return order, nil
}

func (repo *OrderRepository) FindByIDForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) (model.Order, error) {

	var order model.Order

	err := tx.QueryRow(
		ctx,
		findOrderByIDForUpdateQuery,
		id,
	).Scan(
		&order.ID,
		&order.ClientID,
		&order.Status,
		&order.Total,
		&order.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, custom_errors.ErrOrderNotFound
	}

	if err != nil {
		return model.Order{}, fmt.Errorf("find order by id for update: %w", err)
	}

	return order, nil
}

func (repo *OrderRepository) FindAll(
	ctx context.Context,
	limit,
	offset int,
) ([]model.Order, error) {

	rows, err := repo.pool.Query(
		ctx,
		findAllOrdersQuery,
		limit,
		offset,
	)

	if err != nil {
		return nil, fmt.Errorf("find all orders: %w", err)
	}
	defer rows.Close()

	orders := []model.Order{}

	for rows.Next() {

		var order model.Order

		err := rows.Scan(
			&order.ID,
			&order.ClientID,
			&order.Status,
			&order.Total,
			&order.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		orders = append(orders, order)
	}

	return orders, rows.Err()
}

func (repo *OrderRepository) UpdateTotal(
	ctx context.Context,
	tx pgx.Tx,
	orderID uuid.UUID,
	total float64,
) error {

	_, err := tx.Exec(
		ctx,
		updateOrderTotalQuery,
		orderID,
		total,
	)

	if err != nil {
		return fmt.Errorf("update order total: %w", err)
	}

	return nil
}

func (repo *OrderRepository) UpdateStatus(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
	status model.OrderStatus,
) (model.Order, error) {

	var order model.Order

	err := tx.QueryRow(
		ctx,
		updateOrderStatusQuery,
		id,
		status,
	).Scan(
		&order.ID,
		&order.ClientID,
		&order.Status,
		&order.Total,
		&order.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, custom_errors.ErrOrderNotFound
	}

	if err != nil {
		return model.Order{}, fmt.Errorf("update order status: %w", err)
	}

	return order, nil
}
