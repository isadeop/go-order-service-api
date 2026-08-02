package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
	"github.com/isadeop/go-order-service-api/internal/security"
)

type ClientRepository interface {
	Create(ctx context.Context, client model.Client) (model.Client, error)
	FindAll(ctx context.Context) ([]model.Client, error)
	FindByID(ctx context.Context, id uuid.UUID) (model.Client, error)
	FindByEmail(ctx context.Context, email string) (model.Client, error)
}

type ClientService struct {
	repository ClientRepository
}

func NewClientService(repo ClientRepository) *ClientService {
	return &ClientService{
		repository: repo,
	}
}

func (s *ClientService) Create(
	ctx context.Context,
	request dto.CreateClientRequest,
) (dto.ClientResponse, error) {

	if request.Name == "" {
		return dto.ClientResponse{}, custom_errors.ErrClientNameRequired
	}

	if request.Email == "" {
		return dto.ClientResponse{}, custom_errors.ErrClientEmailRequired
	}

	if request.Phone == "" {
		return dto.ClientResponse{}, custom_errors.ErrClientPhoneRequired
	}

	if request.Password == "" {
		return dto.ClientResponse{}, custom_errors.ErrClientPasswordRequired
	}

	_, err := s.repository.FindByEmail(ctx, request.Email)

	if err == nil {
		return dto.ClientResponse{}, custom_errors.ErrClientEmailAlreadyExists
	}

	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		return dto.ClientResponse{}, err
	}

	hash, err := security.HashPassword(request.Password)

	if err != nil {
		return dto.ClientResponse{}, err
	}

	client := model.Client{
		Name:         request.Name,
		Email:        request.Email,
		Phone:        request.Phone,
		PasswordHash: hash,
	}

	client, err = s.repository.Create(ctx, client)

	if err != nil {
		return dto.ClientResponse{}, err
	}

	return dto.NewClientResponse(client), nil
}

func (s *ClientService) FindAll(
	ctx context.Context,
) ([]dto.ClientResponse, error) {

	clients, err := s.repository.FindAll(ctx)

	if err != nil {
		return nil, err
	}

	response := make([]dto.ClientResponse, 0, len(clients))

	for _, client := range clients {
		response = append(
			response,
			dto.NewClientResponse(client),
		)
	}

	return response, nil
}

func (s *ClientService) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (dto.ClientResponse, error) {

	client, err := s.repository.FindByID(ctx, id)

	if err != nil {
		return dto.ClientResponse{}, err
	}

	return dto.NewClientResponse(client), nil
}
