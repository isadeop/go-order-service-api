package domain

import (
	"time"

	"github.com/google/uuid"
)

type StockReservationStatus string

const (
	StockReservationStatusReserved StockReservationStatus = "RESERVED"
	StockReservationStatusReleased StockReservationStatus = "RELEASED"
)

type StockReservation struct {
	SagaID    uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	Status    StockReservationStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
