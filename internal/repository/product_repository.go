package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertProductQuery = `
		INSERT INTO products (name, price, stock)
		VALUES ($1, $2, $3)
		RETURNING id, name, price, stock
	`

	findAllProductsQuery = `
		SELECT id, name, price, stock
		FROM products
		ORDER BY name
	`

	findProductByIDQuery = `
		SELECT id, name, price, stock
		FROM products
		WHERE id = $1
	`

	findProductByNameQuery = `
		SELECT id, name, price, stock
		FROM products
		WHERE name = $1
	`

	findProductByIDForUpdateQuery = `
		SELECT id, name, price, stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`

	updateProductQuery = `
		UPDATE products
		SET
			name = $2,
			price = $3,
			stock = $4,
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, price, stock
	`
	updateProductStockQuery = `
	UPDATE products
	SET
		stock = stock + $2,
		updated_at = now()
	WHERE id = $1 AND stock + $2 >= 0
	`
	deleteProductQuery = `
		DELETE FROM products
		WHERE id = $1
	`
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{
		pool: pool,
	}
}

func (repo *ProductRepository) Create(ctx context.Context, product model.Product) (model.Product, error) {

	err := repo.pool.QueryRow(
		ctx,
		insertProductQuery,
		product.Name,
		product.Price,
		product.Stock,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
	)

	if err != nil {
		return model.Product{}, mapDatabaseErrorProduct("create product", err)
	}

	return product, nil
}

func (repo *ProductRepository) FindAll(ctx context.Context) ([]model.Product, error) {

	rows, err := repo.pool.Query(ctx, findAllProductsQuery)
	if err != nil {
		return nil, fmt.Errorf("find all products: %w", err)
	}
	defer rows.Close()

	products := make([]model.Product, 0)

	for rows.Next() {
		var product model.Product
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Price,
			&product.Stock,
		)

		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

func (repo *ProductRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Product, error) {

	var product model.Product

	err := repo.pool.QueryRow(ctx, findProductByIDQuery, id).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Product{}, custom_errors.ErrProductNotFound
	}

	if err != nil {
		return model.Product{}, fmt.Errorf("find product by id: %w", err)
	}

	return product, nil
}

func (repo *ProductRepository) FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (model.Product, error) {

	var product model.Product

	err := tx.QueryRow(ctx, findProductByIDForUpdateQuery, id).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Product{}, custom_errors.ErrProductNotFound
	}

	if err != nil {
		return model.Product{}, fmt.Errorf("find product by id for update: %w", err)
	}

	return product, nil
}

func (repo *ProductRepository) FindByName(ctx context.Context, name string) (model.Product, error) {

	var product model.Product

	err := repo.pool.QueryRow(ctx, findProductByNameQuery, name).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Product{}, custom_errors.ErrProductNotFound
	}

	if err != nil {
		return model.Product{}, fmt.Errorf("find product by name: %w", err)
	}

	return product, nil
}

func (repo *ProductRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, product model.Product) (model.Product, error) {

	err := tx.QueryRow(
		ctx,
		updateProductQuery,
		id,
		product.Name,
		product.Price,
		product.Stock,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&product.Stock,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Product{}, custom_errors.ErrProductNotFound
	}

	if err != nil {
		return model.Product{}, mapDatabaseErrorProduct("update product", err)
	}

	return product, nil
}

func (repo *ProductRepository) UpdateStock(
	ctx context.Context,
	tx pgx.Tx,
	productID uuid.UUID,
	delta int,
) error {

	commandTag, err := tx.Exec(
		ctx,
		updateProductStockQuery,
		productID,
		delta,
	)

	if err != nil {
		return fmt.Errorf("update product stock: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		if delta < 0 {
			return custom_errors.ErrInsufficientStock
		}
		return custom_errors.ErrProductNotFound
	}

	return nil
}

func (repo *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {

	commandTag, err := repo.pool.Exec(ctx, deleteProductQuery, id)

	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return custom_errors.ErrProductNotFound
	}

	return nil
}

func mapDatabaseErrorProduct(operation string, err error) error {

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {

		switch pgErr.Code {

		case "23505":
			return custom_errors.ErrProductNameExists

		default:
			return fmt.Errorf("%s: %w", operation, err)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}
