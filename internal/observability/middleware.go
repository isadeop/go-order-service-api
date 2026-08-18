package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Middleware registrando cada requisição em formato (JSON)
// incluindo o request id gerado pelo middleware.RequestID do chi
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		result := "ok"
		if ww.Status() >= http.StatusBadRequest {
			result = "error"
		}

		slog.Info("http.request",
			"operation", "http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"result", result,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}
