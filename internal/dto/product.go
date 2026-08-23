package dto

import (
	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

type CreateProductRequest struct {
	Name  string   `json:"name"`
	Price *float64 `json:"price"`
	Stock *int     `json:"stock"`
}

type UpdateProductRequest struct {
	Name  string   `json:"name"`
	Price *float64 `json:"price"`
	Stock *int     `json:"stock"`
}

type ProductResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Price float64   `json:"price"`
	Stock int       `json:"stock"`
}

func NewProductResponse(product domain.Product) ProductResponse {
	return ProductResponse{
		ID:    product.ID,
		Name:  product.Name,
		Price: product.Price,
		Stock: product.Stock,
	}
}
