package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/txport"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertOrderItemQuery = `
		INSERT INTO order_items (order_id, product_id, quantity, price)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	findOrderItemByIDQuery = `
		SELECT id, order_id, product_id, quantity, price
		FROM order_items
		WHERE id = $1
	`

	findItemsByOrderIDQuery = `
		SELECT id, order_id, product_id, quantity, price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`
)

type OrderItemRepository struct {
	pool *pgxpool.Pool
}

func NewOrderItemRepository(pool *pgxpool.Pool) *OrderItemRepository {
	return &OrderItemRepository{
		pool: pool,
	}
}

func (r *OrderItemRepository) Create(
	ctx context.Context,
	tx txport.Tx,
	item domain.OrderItem,
) (domain.OrderItem, error) {

	pTx, err := pgxTx(tx)
	if err != nil {
		return domain.OrderItem{}, err
	}

	err = pTx.QueryRow(
		ctx,
		insertOrderItemQuery,
		item.OrderID,
		item.ProductID,
		item.Quantity,
		item.Price,
	).Scan(
		&item.ID,
	)

	if err != nil {
		return domain.OrderItem{},
			fmt.Errorf("create order item: %w", err)
	}

	return item, nil
}

func (repo *OrderItemRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.OrderItem, error) {

	var item domain.OrderItem

	err := repo.pool.QueryRow(ctx, findOrderItemByIDQuery, id).Scan(
		&item.ID,
		&item.OrderID,
		&item.ProductID,
		&item.Quantity,
		&item.Price,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrderItem{}, custom_errors.ErrOrderItemNotFound
	}

	if err != nil {
		return domain.OrderItem{}, fmt.Errorf("find order of item by id: %w", err)
	}

	return item, nil
}

func (repo *OrderItemRepository) FindByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
) ([]domain.OrderItem, error) {

	rows, err := repo.pool.Query(
		ctx,
		findItemsByOrderIDQuery,
		orderID,
	)

	if err != nil {
		return nil, fmt.Errorf("find order items: %w", err)
	}
	defer rows.Close()

	items := []domain.OrderItem{}

	for rows.Next() {

		var item domain.OrderItem

		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.Price,
		)

		if err != nil {
			return nil, fmt.Errorf("find order item: %w", err)
		}

		items = append(items, item)
	}

	return items, rows.Err()
}
