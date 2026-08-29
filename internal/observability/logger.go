// Centraliza a configuração de log/slog
package observability

import (
	"log/slog"
	"os"
)

// NewLogger cria um logger JSON estruturado
// identifica de qual processo (ex: order-service, stock-service) o log veio
func NewLogger(serviceName string) *slog.Logger {

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler).With(
		slog.String("service", serviceName),
	)
}

// SetDefaultLogger instala o logger estruturado do serviço como o logger
// padrão do pacote log/slog
func SetDefaultLogger(serviceName string) {
	slog.SetDefault(NewLogger(serviceName))
}
