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
	"github.com/isadeop/go-order-service-api/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	observability.SetDefaultLogger()

	ctx := context.Background()

	cfg := config.Load()

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

	connPool := repository.NewConnPool(pool)

	clientRepository :=
		repository.NewClientRepository(pool)

	clientService :=
		application.NewClientService(clientRepository)

	clientController :=
		controllers.NewClientController(clientService)

	productRepository :=
		repository.NewProductRepository(pool)

	productService :=
		application.NewProductService(connPool, productRepository)

	productController :=
		controllers.NewProductController(productService)

	orderRepository :=
		repository.NewOrderRepository(pool)

	orderItemRepository :=
		repository.NewOrderItemRepository(pool)

	productStockRepository :=
		repository.NewProductRepository(pool)

	orderService :=
		application.NewOrderService(
			connPool,
			orderRepository,
			orderItemRepository,
			productStockRepository,
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

	routes.ProductRoutes(
		r,
		productController,
	)

	routes.OrderRoutes(
		r,
		orderController,
	)

	slog.Info("server.starting",
		"operation", "startup",
		"port", cfg.Port,
	)

	slog.Info("server.routes_registered",
		"operation", "startup",
		"routes", []string{
			"POST /clientes",
			"GET /clientes",
			"GET /clientes/{id}",
			"POST /produtos",
			"GET /produtos",
			"GET /produtos/{id}",
			"PUT /produtos/{id}",
			"DELETE /produtos/{id}",
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
