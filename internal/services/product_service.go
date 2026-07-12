package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
	"github.com/isadeop/go-order-service-api/internal/repository"
)

type ProductRepository interface {
	Create(ctx context.Context, client model.Product) (model.Product, error)
	FindAll(ctx context.Context) ([]model.Product, error)
	FindByID(ctx context.Context, id uuid.UUID) (model.Product, error)
	FindByName(ctx context.Context, name string) (model.Product, error)
	Update(ctx context.Context, id uuid.UUID, product model.Product) (model.Product, error)
	UpdateStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID, stock int) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductService struct {
	repository ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{
		repository: repo,
	}
}

func (s *ProductService) Create(
	ctx context.Context,
	request dto.CreateProductRequest,
) (dto.ProductResponse, error) {

	if request.Name == "" {
		return dto.ProductResponse{}, custom_errors.ErrProductNameRequired
	}

	if request.Price == nil {
		return dto.ProductResponse{}, custom_errors.ErrProductPriceRequired
	}

	if *request.Price <= 0 {
		return dto.ProductResponse{}, custom_errors.ErrProductPriceInvalid
	}

	if request.Stock == nil {
		return dto.ProductResponse{}, custom_errors.ErrProductStockRequired
	}

	if *request.Stock < 0 {
		return dto.ProductResponse{}, custom_errors.ErrProductStockInvalid
	}

	_, err := s.repository.FindByName(ctx, request.Name)

	if err == nil {
		return dto.ProductResponse{}, custom_errors.ErrProductNameExists
	}

	if err != nil &&
		!errors.Is(err, custom_errors.ErrProductNotFound) {

		return dto.ProductResponse{}, err
	}

	product := model.Product{
		Name:  request.Name,
		Price: *request.Price,
		Stock: *request.Stock,
	}

	product, err = s.repository.Create(ctx, product)

	if err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.NewProductResponse(product), nil
}

func (s *ProductService) FindAll(
	ctx context.Context,
) ([]dto.ProductResponse, error) {

	products, err := s.repository.FindAll(ctx)

	if err != nil {
		return nil, err
	}

	response := make([]dto.ProductResponse, 0, len(products))

	for _, product := range products {
		response = append(
			response,
			dto.NewProductResponse(product),
		)
	}

	return response, nil
}

func (s *ProductService) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (dto.ProductResponse, error) {

	product, err := s.repository.FindByID(ctx, id)

	if err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.NewProductResponse(product), nil
}

func (s *ProductService) Update(
	ctx context.Context,
	id uuid.UUID,
	request dto.UpdateProductRequest,
) (dto.ProductResponse, error) {

	if request.Name == "" {
		return dto.ProductResponse{}, custom_errors.ErrProductNameRequired
	}

	if request.Price == nil {
		return dto.ProductResponse{}, custom_errors.ErrProductPriceRequired
	}

	if *request.Price <= 0 {
		return dto.ProductResponse{}, custom_errors.ErrProductPriceInvalid
	}

	if request.Stock == nil {
		return dto.ProductResponse{}, custom_errors.ErrProductStockRequired
	}

	if *request.Stock < 0 {
		return dto.ProductResponse{}, custom_errors.ErrProductStockInvalid
	}

	product := model.Product{
		Name:  request.Name,
		Price: *request.Price,
		Stock: *request.Stock,
	}

	product, err := s.repository.Update(
		ctx,
		id,
		product,
	)

	if err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.NewProductResponse(product), nil
}

func (s *ProductService) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return s.repository.Delete(ctx, id)
}
