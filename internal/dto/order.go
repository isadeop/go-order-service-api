package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/model"
)

type CreateOrderRequest struct {
	ClientID uuid.UUID                `json:"client_id"`
	Items    []CreateOrderItemRequest `json:"items"`
}

type OrderResponse struct {
	ID        uuid.UUID           `json:"id"`
	ClientID  uuid.UUID           `json:"client_id"`
	Status    model.OrderStatus   `json:"status"`
	Total     float64             `json:"total"`
	Items     []OrderItemResponse `json:"items,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
}

func NewOrderResponse(order model.Order) OrderResponse {
	return OrderResponse{
		ID:        order.ID,
		ClientID:  order.ClientID,
		Status:    order.Status,
		Total:     order.Total,
		Items:     []OrderItemResponse{},
		CreatedAt: order.CreatedAt,
	}
}
