// stock-service expõe produtos e estoque, e é dono do banco stock_db.

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/isadeop/go-order-service-api/internal/application"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/controllers"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/http/routes"
	"github.com/isadeop/go-order-service-api/internal/entrypoint/messaging/consumers"
	"github.com/isadeop/go-order-service-api/internal/infra/config"
	"github.com/isadeop/go-order-service-api/internal/infra/database"
	"github.com/isadeop/go-order-service-api/internal/infra/messaging"
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

	stockReservationRepository :=
		repository.NewStockReservationRepository(pool)

	productService :=
		application.NewProductService(connPool, productRepository, stockReservationRepository)

	productController :=
		controllers.NewProductController(productService)

	redpandaBrokers := strings.Split(getEnv("REDPANDA_BROKERS", "localhost:19092"), ",")

	commandConsumerClient, err := kgo.NewClient(
		kgo.SeedBrokers(redpandaBrokers...),
		kgo.ConsumeTopics(messaging.TopicStockCommands),
		kgo.ConsumerGroup("stock-service"),
	)
	if err != nil {
		slog.Error("redpanda.client_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}
	defer commandConsumerClient.Close()

	// replyProducerClient publica as respostas (stock-events).
	replyProducerClient, err := kgo.NewClient(kgo.SeedBrokers(redpandaBrokers...))
	if err != nil {
		slog.Error("redpanda.client_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}
	defer replyProducerClient.Close()

	if err := messaging.EnsureTopics(ctx, replyProducerClient, 1, messaging.TopicStockCommands, messaging.TopicStockEvents); err != nil {
		slog.Error("redpanda.ensure_topics_failed",
			"operation", "startup",
			"result", "error",
			"err", err.Error(),
		)
		os.Exit(1)
	}

	reserveConsumer := consumers.NewReserveConsumer(commandConsumerClient, replyProducerClient, messaging.TopicStockEvents, productService)
	go reserveConsumer.Run(ctx)

	// Reconciliação varre periodicamente reservas órfãs
	reservationStaleAfter := getDurationEnv("STOCK_RESERVATION_STALE_AFTER", 10*time.Minute)
	reservationSweepInterval := getDurationEnv("STOCK_RESERVATION_SWEEP_INTERVAL", 1*time.Minute)
	go runReservationReconciliation(ctx, productService, reservationSweepInterval, reservationStaleAfter)

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
		"redpanda_brokers", strings.Join(redpandaBrokers, ","),
		"stock_reservation_stale_after", reservationStaleAfter.String(),
		"stock_reservation_sweep_interval", reservationSweepInterval.String(),
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

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		slog.Warn("config.invalid_duration_env",
			"operation", "startup",
			"key", key,
			"value", value,
			"fallback", fallback.String(),
		)
		return fallback
	}

	return parsed
}

func runReservationReconciliation(ctx context.Context, productService *application.ProductService, interval time.Duration, staleAfter time.Duration) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			released, err := productService.ReconcileStaleReservations(ctx, staleAfter)

			if err != nil {
				slog.Error("stock.reconciliation.sweep_failed",
					"operation", "ReconcileStaleReservations",
					"result", "error",
					"err", err.Error(),
				)
				continue
			}

			if released > 0 {
				slog.Warn("stock.reconciliation.sweep_completed",
					"operation", "ReconcileStaleReservations",
					"result", "ok",
					"released_count", released,
				)
			}
		}
	}
}
