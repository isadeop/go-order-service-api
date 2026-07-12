package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
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

func writeClientError(w http.ResponseWriter, err error) {

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
		log.Printf("internal error: %v", err)
		http.Error(w, "erro interno do servidor", http.StatusInternalServerError)
	}
}

func (c *ClientController) CreateClient(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request dto.CreateClientRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}

	response, err := c.service.Create(
		r.Context(),
		request,
	)

	if err != nil {
		log.Println("erro ao criar cliente:", err)
		writeClientError(w, err)
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
		writeClientError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (c *ClientController) FindClientByID(
	w http.ResponseWriter,
	r *http.Request,
) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		writeClientError(w, custom_errors.ErrInvalidClientID)
		return
	}

	response, err := c.service.FindByID(
		r.Context(),
		id,
	)

	if err != nil {
		writeClientError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}
