// order-service expõe clientes e pedidos e é dono do banco orders_db
// Não acessa mais o banco de produtos diretamente
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/isadeop/go-order-service-api/internal/application"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/controllers"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/routes"
	"github.com/isadeop/go-order-service-api/internal/infra/config"
	"github.com/isadeop/go-order-service-api/internal/infra/database"
	"github.com/isadeop/go-order-service-api/internal/infra/repository"
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

	// connPool adapta *pgxpool.Pool à porta services.ConnPool: é o único
	// ponto em que os casos de uso passam a abrir transações reais de
	// Postgres, sem que internal/application precise importar pgx.
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

	productStock := stockclient.New(stockServiceURL)

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
