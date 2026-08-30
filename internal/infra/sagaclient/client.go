package sagaclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/infra/messaging"
)

// Client é o lado order-service da troca de reserva de estoque.
type Client struct {
	producer      *messaging.Producer
	commandsTopic string
	timeout       time.Duration

	mu      sync.Mutex
	waiters map[uuid.UUID]chan messaging.Envelope // chave: message_id do comando publicado
}

func New(ctx context.Context, producerClient *kgo.Client, consumerClient *kgo.Client, commandsTopic string, timeout time.Duration) *Client {
	c := &Client{
		producer:      messaging.NewProducer(producerClient),
		commandsTopic: commandsTopic,
		timeout:       timeout,
		waiters:       make(map[uuid.UUID]chan messaging.Envelope),
	}

	go c.consumeReplies(ctx, consumerClient)

	return c
}

func (c *Client) consumeReplies(ctx context.Context, client *kgo.Client) {
	for {
		if ctx.Err() != nil {
			return
		}

		fetches := client.PollFetches(ctx)

		if ctx.Err() != nil {
			return
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			slog.Error("order.saga.fetch_error",
				"operation", "consumeReplies",
				"result", "error",
				"topic", topic,
				"partition", partition,
				"err", err.Error(),
			)
		})

		fetches.EachRecord(func(record *kgo.Record) {
			var envelope messaging.Envelope
			if err := json.Unmarshal(record.Value, &envelope); err != nil {
				slog.Error("order.saga.invalid_envelope",
					"operation", "consumeReplies",
					"result", "error",
					"err", err.Error(),
				)
				return
			}
			c.dispatch(envelope)
		})
	}
}

// dispatch entrega a resposta a quem estiver esperando por ela
func (c *Client) dispatch(envelope messaging.Envelope) {
	c.mu.Lock()
	reply, ok := c.waiters[envelope.InReplyTo]
	c.mu.Unlock()

	if !ok {
		return
	}

	select {
	case reply <- envelope:
	default:
	}
}

// Reserve publica o comando de reserva e bloqueia até a resposta chegar ou o timeout expirar
func (c *Client) Reserve(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {

	reply := make(chan messaging.Envelope, 1)

	payload := messaging.ReserveRequestedPayload{
		ProductID: productID,
		Quantity:  quantity,
	}

	messageID, err := c.producer.Publish(
		ctx,
		c.commandsTopic,
		[]byte(productID.String()),
		messaging.TypeStockReserveRequested,
		sagaID,
		uuid.Nil,
		payload,
	)
	if err != nil {
		return fmt.Errorf("sagaclient: publicar comando de reserva: %w", err)
	}

	c.mu.Lock()
	c.waiters[messageID] = reply
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.waiters, messageID)
		c.mu.Unlock()
	}()

	waitCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	select {
	case envelope := <-reply:
		return interpretReserveReply(envelope)
	case <-waitCtx.Done():
		return fmt.Errorf(
			"sagaclient: timeout esperando resposta da reserva de estoque (saga_id=%s, product_id=%s)",
			sagaID, productID,
		)
	}
}

func interpretReserveReply(envelope messaging.Envelope) error {
	switch envelope.Type {
	case messaging.TypeStockReserved:
		return nil

	case messaging.TypeStockReservationFailed:
		var payload messaging.StockReservationFailedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return fmt.Errorf("sagaclient: resposta de falha malformada: %w", err)
		}
		return errorFromReason(payload.Reason)

	default:
		return fmt.Errorf("sagaclient: tipo de resposta inesperado: %s", envelope.Type)
	}
}

// mapeia erros
func errorFromReason(reason string) error {
	switch reason {
	case messaging.ReasonInsufficientStock:
		return custom_errors.ErrInsufficientStock
	case messaging.ReasonProductNotFound:
		return custom_errors.ErrProductNotFound
	case messaging.ReasonInvalidQuantity:
		return custom_errors.ErrOrderItemQuantityInvalid
	default:
		return fmt.Errorf("sagaclient: reserva de estoque falhou: %s", reason)
	}
}
