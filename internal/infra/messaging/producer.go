package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *kgo.Client
}

func NewProducer(client *kgo.Client) *Producer {
	return &Producer{client: client}
}

// Publish serializa payload, monta o envelope (com um novo MessageID) e
// publica em topic. key controla o particionamento — mensagens com a
// mesma key vão para a mesma partição, preservando ordem entre elas.
// Retorna o MessageID gerado.
func (p *Producer) Publish(
	ctx context.Context,
	topic string,
	key []byte,
	messageType string,
	sagaID uuid.UUID,
	inReplyTo uuid.UUID,
	payload any,
) (uuid.UUID, error) {

	body, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("messaging: serializar payload: %w", err)
	}

	envelope := Envelope{
		MessageID:  uuid.New(),
		SagaID:     sagaID,
		InReplyTo:  inReplyTo,
		Type:       messageType,
		OccurredAt: time.Now().UTC(),
		Payload:    body,
	}

	envelopeBody, err := json.Marshal(envelope)
	if err != nil {
		return uuid.Nil, fmt.Errorf("messaging: serializar envelope: %w", err)
	}

	result := p.client.ProduceSync(ctx, &kgo.Record{Topic: topic, Key: key, Value: envelopeBody})
	if err := result.FirstErr(); err != nil {
		return uuid.Nil, fmt.Errorf("messaging: publicar em %s: %w", topic, err)
	}

	return envelope.MessageID, nil
}

// EnsureTopics garante que os tópicos existem, criando os que faltarem.
func EnsureTopics(ctx context.Context, client *kgo.Client, partitions int32, topics ...string) error {
	admin := kadm.NewClient(client)

	for _, topic := range topics {
		_, err := admin.CreateTopic(ctx, partitions, 1, nil, topic)
		if err != nil && !errors.Is(err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("messaging: criar tópico %s: %w", topic, err)
		}
	}

	return nil
}
