package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

func TestStockReservationRepository_FindForUpdate_NaoEncontrada(t *testing.T) {
	pool := newStockTestPool(t)
	repo := NewStockReservationRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = repo.FindForUpdate(context.Background(), tx, uuid.New(), uuid.New())

	if !errors.Is(err, custom_errors.ErrStockReservationNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrStockReservationNotFound)
	}
}

func TestStockReservationRepository_Create_DepoisFindForUpdate(t *testing.T) {
	pool := newStockTestPool(t)
	product := createTestProduct(t, pool, 10)
	repo := NewStockReservationRepository(pool)

	sagaID := uuid.New()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	if err := repo.Create(context.Background(), tx, domain.StockReservation{
		SagaID:    sagaID,
		ProductID: product.ID,
		Quantity:  3,
		Status:    domain.StockReservationStatusReserved,
	}); err != nil {
		tx.Rollback(context.Background())
		t.Fatalf("Create retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	tx, err = pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	found, err := repo.FindForUpdate(context.Background(), tx, sagaID, product.ID)
	if err != nil {
		t.Fatalf("FindForUpdate retornou erro inesperado: %v", err)
	}
	if found.Quantity != 3 {
		t.Errorf("quantity = %d, esperado 3", found.Quantity)
	}
	if found.Status != domain.StockReservationStatusReserved {
		t.Errorf("status = %v, esperado %v", found.Status, domain.StockReservationStatusReserved)
	}
}

func TestStockReservationRepository_UpdateStatus_HappyPath(t *testing.T) {
	pool := newStockTestPool(t)
	product := createTestProduct(t, pool, 10)
	repo := NewStockReservationRepository(pool)

	sagaID := uuid.New()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	if err := repo.Create(context.Background(), tx, domain.StockReservation{
		SagaID:    sagaID,
		ProductID: product.ID,
		Quantity:  3,
		Status:    domain.StockReservationStatusReserved,
	}); err != nil {
		tx.Rollback(context.Background())
		t.Fatalf("setup: falha ao criar reserva: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	tx, err = pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	if err := repo.UpdateStatus(context.Background(), tx, sagaID, product.ID, domain.StockReservationStatusReleased); err != nil {
		tx.Rollback(context.Background())
		t.Fatalf("UpdateStatus retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	tx, err = pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	found, err := repo.FindForUpdate(context.Background(), tx, sagaID, product.ID)
	if err != nil {
		t.Fatalf("FindForUpdate retornou erro inesperado: %v", err)
	}
	if found.Status != domain.StockReservationStatusReleased {
		t.Errorf("status = %v, esperado %v", found.Status, domain.StockReservationStatusReleased)
	}
}

func TestStockReservationRepository_UpdateStatus_NaoEncontrada(t *testing.T) {
	pool := newStockTestPool(t)
	repo := NewStockReservationRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	err = repo.UpdateStatus(context.Background(), tx, uuid.New(), uuid.New(), domain.StockReservationStatusReleased)

	if !errors.Is(err, custom_errors.ErrStockReservationNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrStockReservationNotFound)
	}
}

func TestStockReservationRepository_FindStaleReserved_SoRetornaAntigasEReservadas(t *testing.T) {
	pool := newStockTestPool(t)
	product := createTestProduct(t, pool, 10)
	repo := NewStockReservationRepository(pool)

	staleSagaID := uuid.New()
	freshSagaID := uuid.New()
	oldButReleasedSagaID := uuid.New()

	createAndBackdate := func(sagaID uuid.UUID, quantity int, status domain.StockReservationStatus, backdateBy time.Duration) {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("setup: falha ao abrir transação: %v", err)
		}
		if err := repo.Create(context.Background(), tx, domain.StockReservation{
			SagaID:    sagaID,
			ProductID: product.ID,
			Quantity:  quantity,
			Status:    status,
		}); err != nil {
			tx.Rollback(context.Background())
			t.Fatalf("setup: falha ao criar reserva: %v", err)
		}
		if err := tx.Commit(context.Background()); err != nil {
			t.Fatalf("setup: falha ao commitar: %v", err)
		}

		if backdateBy > 0 {
			_, err := pool.Exec(context.Background(),
				"UPDATE stock_reservations SET created_at = $1 WHERE saga_id = $2 AND product_id = $3",
				time.Now().Add(-backdateBy), sagaID, product.ID,
			)
			if err != nil {
				t.Fatalf("setup: falha ao retroceder created_at: %v", err)
			}
		}
	}

	createAndBackdate(staleSagaID, 3, domain.StockReservationStatusReserved, time.Hour)
	createAndBackdate(freshSagaID, 2, domain.StockReservationStatusReserved, 0)
	createAndBackdate(oldButReleasedSagaID, 1, domain.StockReservationStatusReleased, time.Hour)

	stale, err := repo.FindStaleReserved(context.Background(), time.Now().Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("FindStaleReserved retornou erro inesperado: %v", err)
	}

	foundStale := false
	for _, r := range stale {
		if r.SagaID == freshSagaID {
			t.Errorf("reserva fresca não deveria ter sido retornada: %+v", r)
		}
		if r.SagaID == oldButReleasedSagaID {
			t.Errorf("reserva já liberada não deveria ter sido retornada: %+v", r)
		}
		if r.SagaID == staleSagaID {
			foundStale = true
			if r.Quantity != 3 {
				t.Errorf("quantity = %d, esperado 3", r.Quantity)
			}
		}
	}
	if !foundStale {
		t.Error("esperava encontrar a reserva antiga (stale) no resultado")
	}
}
