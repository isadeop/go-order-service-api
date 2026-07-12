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
	insertClientQuery = `
		INSERT INTO clients (name, email, phone, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, phone, created_at, updated_at
	`

	findAllClientsQuery = `
		SELECT id, name, email, phone, created_at, updated_at
		FROM clients
		ORDER BY name
	`

	findClientByIDQuery = `
		SELECT id, name, email, phone, created_at, updated_at
		FROM clients
		WHERE id = $1
	`

	findClientByEmailQuery = `
		SELECT id, name, email, phone, password_hash, created_at, updated_at
		FROM clients
		WHERE email = $1
	`

	updateClientQuery = `
		UPDATE clients
		SET
			name = $2,
			email = $3,
			phone = $4,
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, email, phone, created_at, updated_at
	`

	deleteClientQuery = `
		DELETE FROM clients
		WHERE id = $1
	`
)

type ClientRepository struct {
	pool *pgxpool.Pool
}

func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{
		pool: pool,
	}
}

func (repo *ClientRepository) Create(ctx context.Context, client model.Client) (model.Client, error) {

	err := repo.pool.QueryRow(
		ctx,
		insertClientQuery,
		client.Name,
		client.Email,
		client.Phone,
		client.PasswordHash,
	).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Phone,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		return model.Client{}, mapDatabaseError("create client", err)
	}

	return client, nil
}

func (repo *ClientRepository) FindAll(ctx context.Context) ([]model.Client, error) {

	rows, err := repo.pool.Query(ctx, findAllClientsQuery)
	if err != nil {
		return nil, fmt.Errorf("find all clients: %w", err)
	}
	defer rows.Close()

	clients := make([]model.Client, 0)

	for rows.Next() {
		var client model.Client

		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Email,
			&client.Phone,
			&client.CreatedAt,
			&client.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}

		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clients: %w", err)
	}

	return clients, nil
}

func (repo *ClientRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Client, error) {

	var client model.Client

	err := repo.pool.QueryRow(ctx, findClientByIDQuery, id).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Phone,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Client{}, custom_errors.ErrClientNotFound
	}

	if err != nil {
		return model.Client{}, fmt.Errorf("find client by id: %w", err)
	}

	return client, nil
}

func (repo *ClientRepository) FindByEmail(ctx context.Context, email string) (model.Client, error) {

	var client model.Client

	err := repo.pool.QueryRow(ctx, findClientByEmailQuery, email).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Phone,
		&client.PasswordHash,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Client{}, custom_errors.ErrClientNotFound
	}

	if err != nil {
		return model.Client{}, fmt.Errorf("find client by email: %w", err)
	}

	return client, nil
}

func (repo *ClientRepository) Update(ctx context.Context, id uuid.UUID, client model.Client) (model.Client, error) {

	err := repo.pool.QueryRow(
		ctx,
		updateClientQuery,
		id,
		client.Name,
		client.Email,
		client.Phone,
	).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Phone,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Client{}, custom_errors.ErrClientNotFound
	}

	if err != nil {
		return model.Client{}, mapDatabaseError("update client", err)
	}

	return client, nil
}

func (repo *ClientRepository) Delete(ctx context.Context, id uuid.UUID) error {

	commandTag, err := repo.pool.Exec(ctx, deleteClientQuery, id)

	if err != nil {
		return fmt.Errorf("delete client: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return custom_errors.ErrClientNotFound
	}

	return nil
}

func mapDatabaseError(operation string, err error) error {

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {

		switch pgErr.Code {

		case "23505":
			return custom_errors.ErrClientEmailAlreadyExists

		default:
			return fmt.Errorf("%s: %w", operation, err)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}
