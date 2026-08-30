// order-service expõe clientes e pedidos e é dono do banco orders_db.
// Consulta e libera estoque via HTTP (internal/infra/stockclient) e reserva estoque via
// Redpanda
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/isadeop/go-order-service-api/internal/application"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/controllers"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/routes"
	"github.com/isadeop/go-order-service-api/internal/infra/config"
	"github.com/isadeop/go-order-service-api/internal/infra/database"
	"github.com/isadeop/go-order-service-api/internal/infra/messaging"
	"github.com/isadeop/go-order-service-api/internal/infra/repository"
	"github.com/isadeop/go-order-service-api/internal/infra/sagaclient"
	"github.com/isadeop/go-order-service-api/internal/infra/stockclient"
	"github.com/isadeop/go-order-service-api/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const serviceName = "order-service"

func main() {

	observability.SetDefaultLogger(serviceName)

	ctx := context.Background()

	cfg := config.Load("orders_db")

	pool, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)

	if err != nil {
		slog.Error("database.connect_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}

	defer pool.Close()

	// connPool adapta *pgxpool.Pool à porta services.ConnPool
	connPool := repository.NewConnPool(pool)

	clientRepository :=
		repository.NewClientRepository(pool)

	clientService :=
		application.NewClientService(clientRepository)

	clientController :=
		controllers.NewClientController(clientService)

	orderRepository :=
		repository.NewOrderRepository(pool)

	orderItemRepository :=
		repository.NewOrderItemRepository(pool)

	stockServiceURL := getEnv("STOCK_SERVICE_URL", "http://localhost:8081")
	httpStockClient := stockclient.New(stockServiceURL)

	redpandaBrokers := strings.Split(getEnv("REDPANDA_BROKERS", "localhost:19092"), ",")

	// producerClient publica os comandos de reserva (stock-commands).
	producerClient, err := kgo.NewClient(kgo.SeedBrokers(redpandaBrokers...))
	if err != nil {
		slog.Error("redpanda.client_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}
	defer producerClient.Close()

	if err := messaging.EnsureTopics(ctx, producerClient, 1, messaging.TopicStockCommands, messaging.TopicStockEvents); err != nil {
		slog.Error("redpanda.ensure_topics_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}

	// replyConsumerClient consome as respostas (stock-events)
	replyConsumerClient, err := kgo.NewClient(
		kgo.SeedBrokers(redpandaBrokers...),
		kgo.ConsumeTopics(messaging.TopicStockEvents),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		slog.Error("redpanda.client_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}
	defer replyConsumerClient.Close()

	// reconciliação por tempo (30s)
	sagaReserveClient := sagaclient.New(ctx, producerClient, replyConsumerClient, messaging.TopicStockCommands, 30*time.Second)

	productStock := &productStockGateway{
		http: httpStockClient,
		saga: sagaReserveClient,
	}

	orderService :=
		application.NewOrderService(
			connPool,
			orderRepository,
			orderItemRepository,
			productStock,
			clientRepository,
		)

	orderController :=
		controllers.NewOrderController(orderService)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(observability.RequestLogger)
	r.Use(middleware.Recoverer)

	routes.ClientRoutes(
		r,
		clientController,
	)

	routes.OrderRoutes(
		r,
		orderController,
	)

	slog.Info("server.starting",
		"operation", "startup",
		"port", cfg.Port,
		"stock_service_url", stockServiceURL,
		"redpanda_brokers", strings.Join(redpandaBrokers, ","),
	)

	slog.Info("server.routes_registered",
		"operation", "startup",
		"routes", []string{
			"POST /clientes",
			"GET /clientes",
			"GET /clientes/{id}",
			"POST /pedidos",
			"GET /pedidos?limit=10&offset=0",
			"GET /pedidos/{id}",
			"PATCH /pedidos/{id}/status",
			"POST /pedidos/{id}/pagar",
			"POST /pedidos/{id}/cancelar",
		},
	)

	if err := http.ListenAndServe(
		":"+cfg.Port,
		r,
	); err != nil {

		slog.Error("server.listen_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type productStockGateway struct {
	http *stockclient.Client
	saga *sagaclient.Client
}

func (g *productStockGateway) FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return g.http.FindByID(ctx, id)
}

func (g *productStockGateway) Reserve(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	return g.saga.Reserve(ctx, sagaID, productID, quantity)
}

func (g *productStockGateway) Release(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	return g.http.Release(ctx, sagaID, productID, quantity)
}
