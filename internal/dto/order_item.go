package dto

import (
	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/model"
)

type CreateOrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  *int      `json:"quantity"`
}

type OrderItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int       `json:"quantity"`
	Price       float64   `json:"price"`
}

func NewOrderItemResponse(orderItem model.OrderItem, productName string) OrderItemResponse {
	return OrderItemResponse{
		ID:          orderItem.ID,
		ProductID:   orderItem.ProductID,
		ProductName: productName,
		Quantity:    orderItem.Quantity,
		Price:       orderItem.Price,
	}
}
