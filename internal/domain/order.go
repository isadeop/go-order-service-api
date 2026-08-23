package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
)

type Order struct {
	ID        uuid.UUID   `json:"id"`
	ClientID  uuid.UUID   `json:"client_id"`
	Status    OrderStatus `json:"status"`
	Total     float64     `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusPaid     OrderStatus = "PAID"
	OrderStatusCanceled OrderStatus = "CANCELED"
)

func (o *Order) Pay() error {

	switch o.Status {
	case OrderStatusPaid:
		return custom_errors.ErrOrderAlreadyPaid
	case OrderStatusCanceled:
		return custom_errors.ErrOrderCannotChangeStatus
	}

	o.Status = OrderStatusPaid

	return nil
}

func (o *Order) Cancel() error {

	switch o.Status {
	case OrderStatusCanceled:
		return custom_errors.ErrOrderAlreadyCanceled
	case OrderStatusPaid:
		return custom_errors.ErrOrderCannotChangeStatus
	}

	o.Status = OrderStatusCanceled

	return nil
}

func (o *Order) ChangeStatus(status OrderStatus) error {

	switch o.Status {
	case OrderStatusPaid:
		return custom_errors.ErrOrderAlreadyPaid
	case OrderStatusCanceled:
		return custom_errors.ErrOrderAlreadyCanceled
	}

	o.Status = status

	return nil
}
