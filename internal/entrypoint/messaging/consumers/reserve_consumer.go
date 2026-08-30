// Package consumers traduz mensagens do Redpanda em chamadas
package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/infra/messaging"
)

type ProductService interface {
	Reserve(ctx context.Context, sagaID uuid.UUID, id uuid.UUID, quantity int) (dto.ProductResponse, error)
}

type ReserveConsumer struct {
	consumer    *kgo.Client
	producer    *messaging.Producer
	eventsTopic string
	service     ProductService
}

func NewReserveConsumer(consumerClient *kgo.Client, producerClient *kgo.Client, eventsTopic string, service ProductService) *ReserveConsumer {
	return &ReserveConsumer{
		consumer:    consumerClient,
		producer:    messaging.NewProducer(producerClient),
		eventsTopic: eventsTopic,
		service:     service,
	}
}

func (c *ReserveConsumer) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		fetches := c.consumer.PollFetches(ctx)

		if ctx.Err() != nil {
			return
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			slog.Error("stock.consumer.fetch_error",
				"operation", "ReserveConsumer",
				"result", "error",
				"topic", topic,
				"partition", partition,
				"err", err.Error(),
			)
		})

		fetches.EachRecord(func(record *kgo.Record) {
			c.handle(ctx, record)
		})
	}
}

func (c *ReserveConsumer) handle(ctx context.Context, record *kgo.Record) {

	var envelope messaging.Envelope
	if err := json.Unmarshal(record.Value, &envelope); err != nil {
		slog.Error("stock.consumer.invalid_envelope",
			"operation", "ReserveConsumer",
			"result", "error",
			"err", err.Error(),
		)
		return
	}

	if envelope.Type != messaging.TypeStockReserveRequested {
		return
	}

	logger := slog.With(
		"operation", "Reserve",
		"saga_id", envelope.SagaID.String(),
	)

	var payload messaging.ReserveRequestedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		logger.Error("stock.consumer.invalid_payload", "result", "error", "err", err.Error())
		return
	}

	logger = logger.With("product_id", payload.ProductID.String(), "quantity", payload.Quantity)

	_, err := c.service.Reserve(ctx, envelope.SagaID, payload.ProductID, payload.Quantity)

	if err != nil {
		logger.Warn("stock.reserve.failed", "result", "error", "err", err.Error())

		failedPayload := messaging.StockReservationFailedPayload{
			ProductID: payload.ProductID,
			Quantity:  payload.Quantity,
			Reason:    reasonFromError(err),
		}

		if _, pubErr := c.producer.Publish(
			ctx,
			c.eventsTopic,
			[]byte(envelope.SagaID.String()),
			messaging.TypeStockReservationFailed,
			envelope.SagaID,
			envelope.MessageID,
			failedPayload,
		); pubErr != nil {
			logger.Error("stock.reserve.reply_failed", "result", "error", "err", pubErr.Error())
		}
		return
	}

	logger.Info("stock.reserve.succeeded", "result", "ok")

	reservedPayload := messaging.StockReservedPayload{
		ProductID: payload.ProductID,
		Quantity:  payload.Quantity,
	}

	if _, pubErr := c.producer.Publish(
		ctx,
		c.eventsTopic,
		[]byte(envelope.SagaID.String()),
		messaging.TypeStockReserved,
		envelope.SagaID,
		envelope.MessageID,
		reservedPayload,
	); pubErr != nil {
		logger.Error("stock.reserve.reply_failed", "result", "error", "err", pubErr.Error())
	}
}

func reasonFromError(err error) string {
	switch {
	case errors.Is(err, custom_errors.ErrInsufficientStock):
		return messaging.ReasonInsufficientStock
	case errors.Is(err, custom_errors.ErrProductNotFound):
		return messaging.ReasonProductNotFound
	case errors.Is(err, custom_errors.ErrOrderItemQuantityInvalid):
		return messaging.ReasonInvalidQuantity
	default:
		return messaging.ReasonInternalError
	}
}
