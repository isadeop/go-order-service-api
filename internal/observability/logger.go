package observability

import (
	"log/slog"
	"os"
)

const ServiceName = "order-service-api"

func NewLogger() *slog.Logger {

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler).With(
		slog.String("service", ServiceName),
	)
}

func SetDefaultLogger() {
	slog.SetDefault(NewLogger())
}
