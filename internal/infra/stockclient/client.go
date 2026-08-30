package stockclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type productResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Price float64   `json:"price"`
	Stock int       `json:"stock"`
}

// FindByID busca um produto no stock-service (GET /produtos/{id}).
func (c *Client) FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/produtos/"+id.String(), nil)
	if err != nil {
		return domain.Product{}, fmt.Errorf("stockclient: montar requisição: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return domain.Product{}, fmt.Errorf("stockclient: chamar stock-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return domain.Product{}, custom_errors.ErrProductNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return domain.Product{}, fmt.Errorf("stockclient: stock-service retornou status %d", resp.StatusCode)
	}

	var body productResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return domain.Product{}, fmt.Errorf("stockclient: decodificar resposta: %w", err)
	}

	return domain.Product{
		ID:    body.ID,
		Name:  body.Name,
		Price: body.Price,
		Stock: body.Stock,
	}, nil
}

type quantityRequest struct {
	Quantity int       `json:"quantity"`
	SagaID   uuid.UUID `json:"saga_id"`
}

// Release chama POST /produtos/{id}/liberar
func (c *Client) Release(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	return c.postQuantity(ctx, "liberar", sagaID, productID, quantity)
}

func (c *Client) postQuantity(ctx context.Context, action string, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {

	payload, err := json.Marshal(quantityRequest{Quantity: quantity, SagaID: sagaID})
	if err != nil {
		return fmt.Errorf("stockclient: montar payload: %w", err)
	}

	url := fmt.Sprintf("%s/produtos/%s/%s", c.baseURL, productID, action)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("stockclient: montar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("stockclient: chamar stock-service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return custom_errors.ErrProductNotFound
	case http.StatusConflict:
		return custom_errors.ErrInsufficientStock
	case http.StatusBadRequest:
		return custom_errors.ErrOrderItemQuantityInvalid
	default:
		return fmt.Errorf("stockclient: stock-service retornou status %d", resp.StatusCode)
	}
}
