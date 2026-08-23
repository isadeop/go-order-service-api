package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

func TestProductRepository_Create_HappyPath(t *testing.T) {
	pool := newTestPool(t)

	product := createTestProduct(t, pool, 10)

	if product.ID == uuid.Nil {
		t.Error("esperava que o ID fosse preenchido pelo banco")
	}
	if product.Price != 10 {
		t.Errorf("price = %v, esperado 10", product.Price)
	}
	if product.Stock != 10 {
		t.Errorf("stock = %d, esperado 10", product.Stock)
	}
}

func TestProductRepository_Create_NomeDuplicado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	existing := createTestProduct(t, pool, 10)

	_, err := repo.Create(context.Background(), domain.Product{
		Name:  existing.Name,
		Price: 20,
		Stock: 5,
	})

	if !errors.Is(err, custom_errors.ErrProductNameExists) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNameExists)
	}
}

func TestProductRepository_Create_EstoqueNegativoViolaConstraint(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	_, err := repo.Create(context.Background(), domain.Product{
		Name:  "Produto Estoque Invalido " + uuid.NewString(),
		Price: 10,
		Stock: -1,
	})

	if err == nil {
		t.Fatal("esperava erro do banco: a constraint CHECK(stock >= 0) deveria rejeitar estoque negativo")
	}
}

func TestProductRepository_FindByID_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}

func TestProductRepository_FindByName_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	created := createTestProduct(t, pool, 10)

	found, err := repo.FindByName(context.Background(), created.Name)
	if err != nil {
		t.Fatalf("FindByName retornou erro inesperado: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, esperado %v", found.ID, created.ID)
	}
}

func TestProductRepository_FindByName_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	_, err := repo.FindByName(context.Background(), "produto-inexistente-"+uuid.NewString())

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}

func TestProductRepository_FindAll_ContemProdutoCriado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	created := createTestProduct(t, pool, 10)

	products, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}

	found := false
	for _, p := range products {
		if p.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("esperava encontrar o produto recém-criado na listagem")
	}
}

func TestProductRepository_Update_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	created := createTestProduct(t, pool, 10)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	updated, err := repo.Update(context.Background(), tx, created.ID, domain.Product{
		Name:  created.Name + " Atualizado",
		Price: 99.9,
		Stock: 3,
	})
	if err != nil {
		t.Fatalf("Update retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	if updated.Price != 99.9 || updated.Stock != 3 {
		t.Errorf("produto atualizado inesperado: %+v", updated)
	}
}

func TestProductRepository_Update_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = repo.Update(context.Background(), tx, uuid.New(), domain.Product{
		Name:  "Nome",
		Price: 10,
		Stock: 1,
	})

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}

func TestProductRepository_UpdateStock_Decrementa(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	created := createTestProduct(t, pool, 10)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	if err := repo.UpdateStock(context.Background(), tx, created.ID, -4); err != nil {
		t.Fatalf("UpdateStock retornou erro inesperado: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("falha ao commitar: %v", err)
	}

	updated, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if updated.Stock != 6 {
		t.Errorf("estoque = %d, esperado 6", updated.Stock)
	}
}

func TestProductRepository_UpdateStock_InsuficienteNaoAplicaDelta(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	created := createTestProduct(t, pool, 3)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("setup: falha ao abrir transação: %v", err)
	}
	defer tx.Rollback(context.Background())

	err = repo.UpdateStock(context.Background(), tx, created.ID, -10)

	if !errors.Is(err, custom_errors.ErrInsufficientStock) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrInsufficientStock)
	}
}

func TestProductRepository_FindByIDForUpdate_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	created := createTestProduct(t, pool, 10)

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

func TestProductRepository_Delete_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	product, err := repo.Create(context.Background(), domain.Product{
		Name:  "Produto Para Deletar " + uuid.NewString(),
		Price: 10,
		Stock: 1,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := repo.Delete(context.Background(), product.ID); err != nil {
		t.Fatalf("Delete retornou erro inesperado: %v", err)
	}

	_, err = repo.FindByID(context.Background(), product.ID)
	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("esperava que o produto não existisse mais após Delete, FindByID retornou: %v", err)
	}
}

func TestProductRepository_Delete_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewProductRepository(pool)

	err := repo.Delete(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}
