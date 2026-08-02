package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/isadeop/go-order-service-api/internal/config"
	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
	"github.com/isadeop/go-order-service-api/internal/repository"
)

func newConcurrencyTestPool(t *testing.T) *pgxpool.Pool {
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

func newIntegrationOrderService(t *testing.T, stock int) (service *OrderService, pool *pgxpool.Pool, clientID, productID uuid.UUID) {
	t.Helper()

	pool = newConcurrencyTestPool(t)

	clientRepo := repository.NewClientRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	itemRepo := repository.NewOrderItemRepository(pool)

	client, err := clientRepo.Create(context.Background(), model.Client{
		Name:         "Cliente Concorrência",
		Email:        "concorrencia-" + uuid.NewString() + "@integracao.local",
		Phone:        "11999999999",
		PasswordHash: "hash-fake-de-teste",
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar cliente: %v", err)
	}

	product, err := productRepo.Create(context.Background(), model.Product{
		Name:  "Produto Concorrência " + uuid.NewString(),
		Price: 10,
		Stock: stock,
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar produto: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE client_id = $1)", client.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM orders WHERE client_id = $1", client.ID)
		_ = productRepo.Delete(ctx, product.ID)
		_ = clientRepo.Delete(ctx, client.ID)
	})

	service = NewOrderService(pool, orderRepo, itemRepo, productRepo, clientRepo)
	return service, pool, client.ID, product.ID
}

func TestOrderService_Concorrencia_CriacaoNaoPermiteEstoqueNegativo(t *testing.T) {
	const initialStock = 10
	const attempts = 30

	service, pool, clientID, productID := newIntegrationOrderService(t, initialStock)

	var successCount int64
	var insufficientCount int64
	var otherErrors int64

	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			request := dto.CreateOrderRequest{
				ClientID: clientID,
				Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(1)}},
			}

			_, err := service.Create(context.Background(), request)

			switch {
			case err == nil:
				atomic.AddInt64(&successCount, 1)
			case errors.Is(err, custom_errors.ErrInsufficientStock):
				atomic.AddInt64(&insufficientCount, 1)
			default:
				atomic.AddInt64(&otherErrors, 1)
				t.Errorf("erro inesperado durante criação concorrente: %v", err)
			}
		}()
	}
	wg.Wait()

	if otherErrors != 0 {
		t.Fatalf("%d chamadas falharam com erro inesperado (diferente de ErrInsufficientStock)", otherErrors)
	}
	if successCount != initialStock {
		t.Errorf("pedidos criados com sucesso = %d, esperado exatamente %d (1 unidade de estoque por pedido, sem lost update)", successCount, int64(initialStock))
	}
	if successCount+insufficientCount != attempts {
		t.Errorf("total de tentativas contabilizadas = %d, esperado %d", successCount+insufficientCount, int64(attempts))
	}

	productRepo := repository.NewProductRepository(pool)
	product, err := productRepo.FindByID(context.Background(), productID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if product.Stock != 0 {
		t.Errorf("estoque final = %d, esperado 0 (estoque nunca deveria ficar negativo nem sobrar por causa de lost update)", product.Stock)
	}
}

func TestOrderService_Concorrencia_CancelamentoNaoEstornaEstoqueDuasVezes(t *testing.T) {
	const initialStock = 100
	const orderedQuantity = 10
	const cancelAttempts = 10

	service, pool, clientID, productID := newIntegrationOrderService(t, initialStock)

	created, err := service.Create(context.Background(), dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(orderedQuantity)}},
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar pedido: %v", err)
	}

	var successCount int64
	var alreadyCanceledCount int64
	var otherErrors int64

	var wg sync.WaitGroup
	for i := 0; i < cancelAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := service.Cancel(context.Background(), created.ID)

			switch {
			case err == nil:
				atomic.AddInt64(&successCount, 1)
			case errors.Is(err, custom_errors.ErrOrderAlreadyCanceled):
				atomic.AddInt64(&alreadyCanceledCount, 1)
			default:
				atomic.AddInt64(&otherErrors, 1)
				t.Errorf("erro inesperado durante cancelamento concorrente: %v", err)
			}
		}()
	}
	wg.Wait()

	if otherErrors != 0 {
		t.Fatalf("%d cancelamentos falharam com erro inesperado", otherErrors)
	}
	if successCount != 1 {
		t.Errorf("cancelamentos bem-sucedidos = %d, esperado exatamente 1", successCount)
	}
	if alreadyCanceledCount != cancelAttempts-1 {
		t.Errorf("cancelamentos rejeitados como já cancelado = %d, esperado %d", alreadyCanceledCount, int64(cancelAttempts-1))
	}

	productRepo := repository.NewProductRepository(pool)
	product, err := productRepo.FindByID(context.Background(), productID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if product.Stock != initialStock {
		t.Errorf("estoque final = %d, esperado %d (o estorno deve acontecer exatamente uma vez)", product.Stock, initialStock)
	}
}
