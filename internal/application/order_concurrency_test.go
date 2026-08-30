package application

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/controllers"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/routes"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/messaging/consumers"
	"github.com/isadeop/go-order-service-api/internal/infra/config"
	"github.com/isadeop/go-order-service-api/internal/infra/messaging"
	"github.com/isadeop/go-order-service-api/internal/infra/repository"
	"github.com/isadeop/go-order-service-api/internal/infra/sagaclient"
	"github.com/isadeop/go-order-service-api/internal/infra/stockclient"
)

func newConcurrencyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg := config.Load("orders_db")

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		t.Skipf("não foi possível conectar ao postgres de teste (orders_db): %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("postgres de teste indisponível (orders_db): %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func newStockConcurrencyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	database := config.DatabaseConfig{
		Host:     envOrDefault("STOCK_POSTGRES_HOST", "localhost"),
		Port:     envOrDefault("STOCK_POSTGRES_PORT", "5433"),
		User:     envOrDefault("STOCK_POSTGRES_USER", "adm"),
		Password: envOrDefault("STOCK_POSTGRES_PASSWORD", "adm"),
		Name:     envOrDefault("STOCK_POSTGRES_DB", "stock_db"),
		SSLMode:  envOrDefault("STOCK_POSTGRES_SSLMODE", "disable"),
	}

	pool, err := pgxpool.New(context.Background(), database.URL())
	if err != nil {
		t.Skipf("não foi possível conectar ao postgres de teste (stock_db): %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("postgres de teste indisponível (stock_db): %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func redpandaTestBrokers() []string {
	return strings.Split(envOrDefault("REDPANDA_BROKERS", "localhost:19092"), ",")
}

func newRedpandaTestClient(t *testing.T, opts ...kgo.Opt) *kgo.Client {
	t.Helper()

	client, err := kgo.NewClient(append([]kgo.Opt{kgo.SeedBrokers(redpandaTestBrokers()...)}, opts...)...)
	if err != nil {
		t.Skipf("não foi possível criar cliente do redpanda de teste: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx); err != nil {
		client.Close()
		t.Skipf("redpanda de teste indisponível: %v", err)
	}

	t.Cleanup(client.Close)

	return client
}

func newStockServiceTestServer(t *testing.T, stockPool *pgxpool.Pool) *httptest.Server {
	t.Helper()

	stockConnPool := repository.NewConnPool(stockPool)
	productRepository := repository.NewProductRepository(stockPool)
	stockReservationRepository := repository.NewStockReservationRepository(stockPool)
	productService := NewProductService(stockConnPool, productRepository, stockReservationRepository)
	productController := controllers.NewProductController(productService)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	routes.ProductRoutes(r, productController)

	server := httptest.NewServer(r)
	t.Cleanup(server.Close)

	return server
}

type testProductStockGateway struct {
	http *stockclient.Client
	saga *sagaclient.Client
}

func (g *testProductStockGateway) FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return g.http.FindByID(ctx, id)
}

func (g *testProductStockGateway) Reserve(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	return g.saga.Reserve(ctx, sagaID, productID, quantity)
}

func (g *testProductStockGateway) Release(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	return g.http.Release(ctx, sagaID, productID, quantity)
}

func newSagaTestGateway(t *testing.T, stockServer *httptest.Server, productService *ProductService) *testProductStockGateway {
	t.Helper()

	suffix := uuid.NewString()
	commandsTopic := "stock-commands-test-" + suffix
	eventsTopic := "stock-events-test-" + suffix

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	setupClient := newRedpandaTestClient(t)
	if err := messaging.EnsureTopics(context.Background(), setupClient, 1, commandsTopic, eventsTopic); err != nil {
		t.Fatalf("setup: falha ao criar tópicos de teste: %v", err)
	}

	orderProducerClient := newRedpandaTestClient(t)
	orderReplyConsumerClient := newRedpandaTestClient(t,
		kgo.ConsumeTopics(eventsTopic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	sagaReserveClient := sagaclient.New(ctx, orderProducerClient, orderReplyConsumerClient, commandsTopic, 10*time.Second)

	stockCommandConsumerClient := newRedpandaTestClient(t,
		kgo.ConsumeTopics(commandsTopic),
		kgo.ConsumerGroup("stock-service-test-"+suffix),
	)
	stockReplyProducerClient := newRedpandaTestClient(t)
	reserveConsumer := consumers.NewReserveConsumer(stockCommandConsumerClient, stockReplyProducerClient, eventsTopic, productService)
	go reserveConsumer.Run(ctx)

	return &testProductStockGateway{
		http: stockclient.New(stockServer.URL),
		saga: sagaReserveClient,
	}
}

func newIntegrationOrderService(t *testing.T, stock int) (service *OrderService, stockPool *pgxpool.Pool, clientID, productID uuid.UUID) {
	t.Helper()

	ordersPool := newConcurrencyTestPool(t)
	stockPool = newStockConcurrencyTestPool(t)

	clientRepo := repository.NewClientRepository(ordersPool)
	orderRepo := repository.NewOrderRepository(ordersPool)
	itemRepo := repository.NewOrderItemRepository(ordersPool)

	stockConnPool := repository.NewConnPool(stockPool)
	stockProductRepo := repository.NewProductRepository(stockPool)
	stockReservationRepo := repository.NewStockReservationRepository(stockPool)
	stockProductService := NewProductService(stockConnPool, stockProductRepo, stockReservationRepo)

	stockServer := newStockServiceTestServer(t, stockPool)
	productStock := newSagaTestGateway(t, stockServer, stockProductService)

	client, err := clientRepo.Create(context.Background(), domain.Client{
		Name:         "Cliente Concorrência",
		Email:        "concorrencia-" + uuid.NewString() + "@integracao.local",
		Phone:        "11999999999",
		PasswordHash: "hash-fake-de-teste",
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar cliente: %v", err)
	}

	product, err := stockProductRepo.Create(context.Background(), domain.Product{
		Name:  "Produto Concorrência " + uuid.NewString(),
		Price: 10,
		Stock: stock,
	})
	if err != nil {
		t.Fatalf("setup: falha ao criar produto: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = ordersPool.Exec(ctx, "DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE client_id = $1)", client.ID)
		_, _ = ordersPool.Exec(ctx, "DELETE FROM orders WHERE client_id = $1", client.ID)
		_ = stockProductRepo.Delete(ctx, product.ID)
		_ = clientRepo.Delete(ctx, client.ID)
	})

	service = NewOrderService(repository.NewConnPool(ordersPool), orderRepo, itemRepo, productStock, clientRepo)
	return service, stockPool, client.ID, product.ID
}

func TestOrderService_Concorrencia_CriacaoNaoPermiteEstoqueNegativo(t *testing.T) {
	const initialStock = 10
	const attempts = 30

	service, stockPool, clientID, productID := newIntegrationOrderService(t, initialStock)

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

	productRepo := repository.NewProductRepository(stockPool)
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

	service, stockPool, clientID, productID := newIntegrationOrderService(t, initialStock)

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

	productRepo := repository.NewProductRepository(stockPool)
	product, err := productRepo.FindByID(context.Background(), productID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if product.Stock != initialStock {
		t.Errorf("estoque final = %d, esperado %d (o estorno deve acontecer exatamente uma vez)", product.Stock, initialStock)
	}
}
