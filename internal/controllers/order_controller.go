package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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

func writeOrderError(w http.ResponseWriter, operation string, err error) {

	switch {
	case errors.Is(err, custom_errors.ErrInvalidOrderID):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrOrderNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, custom_errors.ErrOrderClientRequired),
		errors.Is(err, custom_errors.ErrOrderItemsRequired),
		errors.Is(err, custom_errors.ErrOrderItemProductRequired),
		errors.Is(err, custom_errors.ErrOrderItemQuantityRequired),
		errors.Is(err, custom_errors.ErrOrderItemQuantityInvalid),
		errors.Is(err, custom_errors.ErrInvalidPagination):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrOrderClientNotFound),
		errors.Is(err, custom_errors.ErrOrderProductNotFound):
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
		slog.Error("order.internal_error",
			"operation", operation,
			"result", "error",
			"err", err.Error(),
		)
		http.Error(w, "erro interno do servidor", http.StatusInternalServerError)
	}
}

func (c *OrderController) CreateOrder(w http.ResponseWriter, r *http.Request) {

	var request dto.CreateOrderRequest

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.Create(r.Context(), request)
	if err != nil {
		writeOrderError(w, "CreateOrder", err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) FindOrderByID(w http.ResponseWriter, r *http.Request) {

	id, err := parseIDParam(r)
	if err != nil {
		writeOrderError(w, "FindOrderByID", custom_errors.ErrInvalidOrderID)
		return
	}

	response, err := c.service.FindByID(r.Context(), id)
	if err != nil {
		writeOrderError(w, "FindOrderByID", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) FindOrders(w http.ResponseWriter, r *http.Request) {

	limit := 10
	offset := 0

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			writeOrderError(w, "FindOrders", custom_errors.ErrInvalidPagination)
			return
		}
		limit = parsed
	}

	if value := r.URL.Query().Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			writeOrderError(w, "FindOrders", custom_errors.ErrInvalidPagination)
			return
		}
		offset = parsed
	}

	response, err := c.service.FindAll(r.Context(), limit, offset)
	if err != nil {
		writeOrderError(w, "FindOrders", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {

	id, err := parseIDParam(r)
	if err != nil {
		writeOrderError(w, "UpdateOrderStatus", custom_errors.ErrInvalidOrderID)
		return
	}

	var request struct {
		Status model.OrderStatus `json:"status"`
	}

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.UpdateStatus(r.Context(), id, request.Status)
	if err != nil {
		writeOrderError(w, "UpdateOrderStatus", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) Pay(w http.ResponseWriter, r *http.Request) {

	id, err := parseIDParam(r)
	if err != nil {
		writeOrderError(w, "Pay", custom_errors.ErrInvalidOrderID)
		return
	}

	response, err := c.service.Pay(r.Context(), id)
	if err != nil {
		writeOrderError(w, "Pay", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *OrderController) Cancel(w http.ResponseWriter, r *http.Request) {

	id, err := parseIDParam(r)
	if err != nil {
		writeOrderError(w, "Cancel", custom_errors.ErrInvalidOrderID)
		return
	}

	response, err := c.service.Cancel(r.Context(), id)
	if err != nil {
		writeOrderError(w, "Cancel", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
