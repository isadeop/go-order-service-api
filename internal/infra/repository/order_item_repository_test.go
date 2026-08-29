package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

// order_items não tem mais FK para products (bancos separados) — o
// product_id é só um UUID validado pela aplicação, não pelo banco.

func TestOrderItemRepository_Create_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	order := createTestOrder(t, pool, client.ID, 0)
	productID := uuid.New()
	repo := NewOrderItemRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	item, err := repo.Create(context.Background(), tx, domain.OrderItem{
		OrderID:   order.ID,
		ProductID: productID,
		Quantity:  2,
		Price:     10,
	})
	if err != nil {
		t.Fatalf("Create retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	if item.ID == uuid.Nil {
		t.Error("esperava que o ID fosse preenchido pelo banco")
	}
}

func TestOrderItemRepository_Create_PedidoInexistenteViolaFK(t *testing.T) {
	pool := newTestPool(t)
	productID := uuid.New()
	repo := NewOrderItemRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = repo.Create(context.Background(), tx, domain.OrderItem{
		OrderID:   uuid.New(), // pedido que não existe
		ProductID: productID,
		Quantity:  1,
		Price:     10,
	})

	if err == nil {
		t.Fatal("esperava erro do banco: a constraint de FK para orders deveria rejeitar order_id inexistente")
	}
}

func TestOrderItemRepository_Create_QuantidadeInvalidaViolaConstraint(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	order := createTestOrder(t, pool, client.ID, 0)
	productID := uuid.New()
	repo := NewOrderItemRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = repo.Create(context.Background(), tx, domain.OrderItem{
		OrderID:   order.ID,
		ProductID: productID,
		Quantity:  0,
		Price:     10,
	})

	if err == nil {
		t.Fatal("esperava erro do banco: a constraint CHECK(quantity > 0) deveria rejeitar quantidade zero")
	}
}

func TestOrderItemRepository_FindByID_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewOrderItemRepository(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrOrderItemNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderItemNotFound)
	}
}

func TestOrderItemRepository_FindByOrderID_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	order := createTestOrder(t, pool, client.ID, 0)
	productA := uuid.New()
	productB := uuid.New()
	repo := NewOrderItemRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}

	if _, err := repo.Create(context.Background(), tx, domain.OrderItem{OrderID: order.ID, ProductID: productA, Quantity: 1, Price: 10}); err != nil {
		tx.Rollback(context.Background())
		t.Fatalf("setup: falha ao criar item: %v", err)
	}
	if _, err := repo.Create(context.Background(), tx, domain.OrderItem{OrderID: order.ID, ProductID: productB, Quantity: 2, Price: 20}); err != nil {
		tx.Rollback(context.Background())
		t.Fatalf("setup: falha ao criar item: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	items, err := repo.FindByOrderID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByOrderID retornou erro inesperado: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("esperava 2 itens, obteve %d", len(items))
	}
}

func TestOrderItemRepository_FindByOrderID_PedidoSemItens(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	order := createTestOrder(t, pool, client.ID, 0)
	repo := NewOrderItemRepository(pool)

	items, err := repo.FindByOrderID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByOrderID retornou erro inesperado: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("esperava 0 itens, obteve %d", len(items))
	}
}
