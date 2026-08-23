package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

func TestOrderRepository_Create_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)

	order := createTestOrder(t, pool, client.ID, 100)

	if order.ID == uuid.Nil {
		t.Error("esperava que o ID fosse preenchido pelo banco")
	}
	if order.ClientID != client.ID {
		t.Errorf("client_id = %v, esperado %v", order.ClientID, client.ID)
	}
	if order.Status != domain.OrderStatusPending {
		t.Errorf("status = %v, esperado %v", order.Status, domain.OrderStatusPending)
	}
	if order.Total != 100 {
		t.Errorf("total = %v, esperado 100", order.Total)
	}
	if order.CreatedAt.IsZero() {
		t.Error("esperava created_at preenchido pelo banco")
	}
}

func TestOrderRepository_Create_ClienteInexistenteViolaFK(t *testing.T) {
	pool := newTestPool(t)
	repo := NewOrderRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = repo.Create(context.Background(), tx, domain.Order{
		ClientID: uuid.New(), // cliente que não existe
		Status:   domain.OrderStatusPending,
		Total:    10,
	})

	if err == nil {
		t.Fatal("esperava erro do banco: a constraint de FK para clients deveria rejeitar client_id inexistente")
	}
}

func TestOrderRepository_FindByID_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewOrderRepository(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrOrderNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderNotFound)
	}
}

func TestOrderRepository_FindByID_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	created := createTestOrder(t, pool, client.ID, 50)
	repo := NewOrderRepository(pool)

	found, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if found.ClientID != client.ID {
		t.Errorf("client_id = %v, esperado %v", found.ClientID, client.ID)
	}
}

func TestOrderRepository_FindByIDForUpdate_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	created := createTestOrder(t, pool, client.ID, 50)
	repo := NewOrderRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	found, err := repo.FindByIDForUpdate(context.Background(), tx, created.ID)
	if err != nil {
		t.Fatalf("FindByIDForUpdate retornou erro inesperado: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, esperado %v", found.ID, created.ID)
	}
}

func TestOrderRepository_FindAll_RespeitaLimiteEContemPedidoCriado(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	repo := NewOrderRepository(pool)

	first := createTestOrder(t, pool, client.ID, 10)
	second := createTestOrder(t, pool, client.ID, 20)
	third := createTestOrder(t, pool, client.ID, 30)

	limited, err := repo.FindAll(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("esperava exatamente 2 pedidos respeitando o limit, obteve %d", len(limited))
	}

	all, err := repo.FindAll(context.Background(), 10000, 0)
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}

	ids := make(map[uuid.UUID]bool, len(all))
	for _, o := range all {
		ids[o.ID] = true
	}
	for _, wantID := range []uuid.UUID{first.ID, second.ID, third.ID} {
		if !ids[wantID] {
			t.Errorf("esperava encontrar o pedido %v na listagem completa", wantID)
		}
	}
}

func TestOrderRepository_UpdateTotal_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	created := createTestOrder(t, pool, client.ID, 0)
	repo := NewOrderRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	if err := repo.UpdateTotal(context.Background(), tx, created.ID, 123.45); err != nil {
		t.Fatalf("UpdateTotal retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	found, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if found.Total != 123.45 {
		t.Errorf("total = %v, esperado 123.45", found.Total)
	}
}

func TestOrderRepository_UpdateStatus_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	client := createTestClient(t, pool)
	created := createTestOrder(t, pool, client.ID, 10)
	repo := NewOrderRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	updated, err := repo.UpdateStatus(context.Background(), tx, created.ID, domain.OrderStatusPaid)
	if err != nil {
		t.Fatalf("UpdateStatus retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	if updated.Status != domain.OrderStatusPaid {
		t.Errorf("status = %v, esperado %v", updated.Status, domain.OrderStatusPaid)
	}
}

func TestOrderRepository_UpdateStatus_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewOrderRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = repo.UpdateStatus(context.Background(), tx, uuid.New(), domain.OrderStatusPaid)

	if !errors.Is(err, custom_errors.ErrOrderNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderNotFound)
	}
}
