// Package messaging contém o contrato de mensagens da Saga de criação de pedido
package messaging

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TopicStockCommands = "stock-commands"
	TopicStockEvents   = "stock-events"
)

const (
	TypeStockReserveRequested  = "stock.reserve.requested"
	TypeStockReserved          = "stock.reserved"
	TypeStockReservationFailed = "stock.reservation.failed"
)

// Motivos de falha
const (
	ReasonInsufficientStock = "insufficient_stock"
	ReasonProductNotFound   = "product_not_found"
	ReasonInvalidQuantity   = "invalid_quantity"
	ReasonInternalError     = "internal_error"
)

// Envelope é o formato comum a toda mensagem da Saga
type Envelope struct {
	MessageID  uuid.UUID       `json:"message_id"`
	SagaID     uuid.UUID       `json:"saga_id"`
	InReplyTo  uuid.UUID       `json:"in_reply_to,omitempty"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// ReserveRequestedPayload é o payload de stock.reserve.requested.
type ReserveRequestedPayload struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// StockReservedPayload é o payload de stock.reserved.
type StockReservedPayload struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// StockReservationFailedPayload é o payload de stock.reservation.failed.
type StockReservationFailedPayload struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Reason    string    `json:"reason"`
}
