package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// parseIDParam extrai o parâmetro de rota "id" e converte para uuid.UUID.
// utilizado em todos os controllers
func parseIDParam(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}
