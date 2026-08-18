package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
)

type Product struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p *Product) Reserve(quantity int) error {

	if quantity <= 0 {
		return custom_errors.ErrOrderItemQuantityInvalid
	}

	if quantity > p.Stock {
		return custom_errors.ErrInsufficientStock
	}

	p.Stock -= quantity

	return nil
}

func (p *Product) Release(quantity int) {
	p.Stock += quantity
}
