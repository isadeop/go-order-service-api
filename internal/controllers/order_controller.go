package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
)

type OrderService interface {
	Create(ctx context.Context, request dto.CreateOrderRequest) (dto.OrderResponse, error)
	FindByID(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error)
	FindAll(ctx context.Context, limit int, offset int) ([]dto.OrderResponse, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) (dto.OrderResponse, error)
	Pay(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error)
	Cancel(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error)
}

type OrderController struct {
	service OrderService
}

func NewOrderController(service OrderService) *OrderController {
	return &OrderController{service: service}
}

func writeOrderError(w http.ResponseWriter, err error) {

	switch {
	case errors.Is(err, custom_errors.ErrInvalidOrderID):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrOrderNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, custom_errors.ErrOrderClientRequired),
		errors.Is(err, custom_errors.ErrOrderItemsRequired),
		errors.Is(err, custom_errors.ErrOrderProductNotFound),
		errors.Is(err, custom_errors.ErrOrderItemProductRequired),
		errors.Is(err, custom_errors.ErrOrderItemQuantityRequired),
		errors.Is(err, custom_errors.ErrOrderItemQuantityInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrOrderClientNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, custom_errors.ErrInsufficientStock):
		http.Error(w, err.Error(), http.StatusConflict)

	case errors.Is(err, custom_errors.ErrOrderAlreadyPaid),
		errors.Is(err, custom_errors.ErrOrderAlreadyCanceled),
		errors.Is(err, custom_errors.ErrOrderCannotChangeStatus):
		http.Error(w, err.Error(), http.StatusConflict)

	case errors.Is(err, custom_errors.ErrInvalidOrderStatus):
		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		log.Printf("internal order error: %v", err)
		http.Error(w, "erro interno do servidor", http.StatusInternalServerError)
	}
}

func (c *OrderController) CreateOrder(w http.ResponseWriter, r *http.Request) {

	var request dto.CreateOrderRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}

	response, err := c.service.Create(r.Context(), request)
	if err != nil {
		writeOrderError(w, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) FindOrderByID(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeOrderError(w, custom_errors.ErrInvalidOrderID)
		return
	}

	response, err := c.service.FindByID(r.Context(), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) FindOrders(w http.ResponseWriter, r *http.Request) {

	limit := 10
	offset := 0

	if value := r.URL.Query().Get("limit"); value != "" {
		fmt.Sscanf(value, "%d", &limit)
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		fmt.Sscanf(value, "%d", &offset)
	}

	response, err := c.service.FindAll(r.Context(), limit, offset)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeOrderError(w, custom_errors.ErrInvalidOrderID)
		return
	}

	var request struct {
		Status model.OrderStatus `json:"status"`
	}

	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}

	response, err := c.service.UpdateStatus(r.Context(), id, request.Status)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) Pay(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeOrderError(w, custom_errors.ErrInvalidOrderID)
		return
	}

	response, err := c.service.Pay(r.Context(), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) Cancel(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeOrderError(w, custom_errors.ErrInvalidOrderID)
		return
	}

	response, err := c.service.Cancel(r.Context(), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
