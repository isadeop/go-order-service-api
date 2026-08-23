package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type ClientService interface {
	Create(ctx context.Context, request dto.CreateClientRequest) (dto.ClientResponse, error)
	FindAll(ctx context.Context) ([]dto.ClientResponse, error)
	FindByID(ctx context.Context, id uuid.UUID) (dto.ClientResponse, error)
}

type ClientController struct {
	service ClientService
}

func NewClientController(service ClientService) *ClientController {
	return &ClientController{
		service: service,
	}
}

func writeClientError(w http.ResponseWriter, operation string, err error) {

	switch {

	case errors.Is(err, custom_errors.ErrInvalidClientID):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrClientNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, custom_errors.ErrClientEmailAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)

	case errors.Is(err, custom_errors.ErrClientNameRequired),
		errors.Is(err, custom_errors.ErrClientEmailRequired),
		errors.Is(err, custom_errors.ErrClientPhoneRequired),
		errors.Is(err, custom_errors.ErrClientPasswordRequired):

		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		slog.Error("client.internal_error",
			"operation", operation,
			"result", "error",
			"err", err.Error(),
		)
		http.Error(w, "erro interno do servidor", http.StatusInternalServerError)
	}
}

func (c *ClientController) CreateClient(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request dto.CreateClientRequest

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.Create(
		r.Context(),
		request,
	)

	if err != nil {
		slog.Warn("client.create_failed",
			"operation", "CreateClient",
			"result", "error",
			"err", err.Error(),
		)
		writeClientError(w, "CreateClient", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

func (c *ClientController) FindAllClients(
	w http.ResponseWriter,
	r *http.Request,
) {

	response, err := c.service.FindAll(r.Context())

	if err != nil {
		writeClientError(w, "FindAllClients", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (c *ClientController) FindClientByID(
	w http.ResponseWriter,
	r *http.Request,
) {

	id, err := parseIDParam(r)

	if err != nil {
		writeClientError(w, "FindClientByID", custom_errors.ErrInvalidClientID)
		return
	}

	response, err := c.service.FindByID(
		r.Context(),
		id,
	)

	if err != nil {
		writeClientError(w, "FindClientByID", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}
