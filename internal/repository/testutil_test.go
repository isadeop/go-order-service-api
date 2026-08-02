package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/isadeop/go-order-service-api/internal/config"
	"github.com/isadeop/go-order-service-api/internal/model"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		t.Skipf("não foi possível conectar ao postgres de teste: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("postgres de teste indisponível: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

// createTestClient insere um cliente com e-mail único e faz sua remoção ao final do teste.
func createTestClient(t *testing.T, pool *pgxpool.Pool) model.Client {
	t.Helper()

	repo := NewClientRepository(pool)

	client, err := repo.Create(context.Background(), model.Client{
		Name:         "Cliente de Teste de Integração",
		Email:        "teste-" + uuid.NewString() + "@integracao.local",
		Phone:        "11999999999",
		PasswordHash: "hash-fake-de-teste",
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar cliente de teste: %v", err)
	}

	// remove primeiro os itens do pedido e o pedido para depois remover o cliente
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE client_id = $1)", client.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM orders WHERE client_id = $1", client.ID)
		_ = repo.Delete(ctx, client.ID)
	})

	return client
}

// createTestProduct insere um produto com nome único e faz sua remoção ao final do teste
func createTestProduct(t *testing.T, pool *pgxpool.Pool, stock int) model.Product {
	t.Helper()

	repo := NewProductRepository(pool)

	product, err := repo.Create(context.Background(), model.Product{
		Name:  "Produto de Teste de Integração " + uuid.NewString(),
		Price: 10,
		Stock: stock,
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar produto de teste: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM order_items WHERE product_id = $1", product.ID)
		_ = repo.Delete(ctx, product.ID)
	})

	return product
}

func createTestOrder(t *testing.T, pool *pgxpool.Pool, clientID uuid.UUID, total float64) model.Order {
	t.Helper()

	repo := NewOrderRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	order, err := repo.Create(context.Background(), tx, model.Order{
		ClientID: clientID,
		Status:   model.OrderStatusPending,
		Total:    total,
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar pedido de teste: %v", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("setup: falha ao commitar pedido de teste: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM order_items WHERE order_id = $1", order.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", order.ID)
	})

	return order
}
