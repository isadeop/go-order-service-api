package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

type CreateOrderRequest struct {
	ClientID uuid.UUID                `json:"client_id"`
	Items    []CreateOrderItemRequest `json:"items"`
}

type OrderResponse struct {
	ID         uuid.UUID           `json:"id"`
	ClientID   uuid.UUID           `json:"client_id"`
	ClientName string              `json:"client_name"`
	Status     domain.OrderStatus  `json:"status"`
	Total      float64             `json:"total"`
	Items      []OrderItemResponse `json:"items,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
}

func NewOrderResponse(order domain.Order, clientName string) OrderResponse {
	return OrderResponse{
		ID:         order.ID,
		ClientID:   order.ClientID,
		ClientName: clientName,
		Status:     order.Status,
		Total:      order.Total,
		Items:      []OrderItemResponse{},
		CreatedAt:  order.CreatedAt,
	}
}
