// stock-service expõe produtos e estoque, e é dono do banco stock_db.

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

const serviceName = "stock-service"

func main() {

	observability.SetDefaultLogger(serviceName)

	ctx := context.Background()

	cfg := config.Load("stock_db")

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

	productRepository :=
		repository.NewProductRepository(pool)

	productService :=
		application.NewProductService(connPool, productRepository)

	productController :=
		controllers.NewProductController(productService)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(observability.RequestLogger)
	r.Use(middleware.Recoverer)

	routes.ProductRoutes(
		r,
		productController,
	)

	slog.Info("server.starting",
		"operation", "startup",
		"port", cfg.Port,
	)

	slog.Info("server.routes_registered",
		"operation", "startup",
		"routes", []string{
			"POST /produtos",
			"GET /produtos",
			"GET /produtos/{id}",
			"PUT /produtos/{id}",
			"DELETE /produtos/{id}",
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
