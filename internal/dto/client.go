package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

type CreateClientRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type ClientResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Update mantido para melhoria futura
type UpdateClientRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func NewClientResponse(client domain.Client) ClientResponse {
	return ClientResponse{
		ID:        client.ID,
		Name:      client.Name,
		Email:     client.Email,
		Phone:     client.Phone,
		CreatedAt: client.CreatedAt,
		UpdatedAt: client.UpdatedAt,
	}
}
