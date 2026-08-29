package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type ProductRepository interface {
	Create(ctx context.Context, client domain.Product) (domain.Product, error)
	FindAll(ctx context.Context) ([]domain.Product, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	FindByIDForUpdate(ctx context.Context, tx Tx, id uuid.UUID) (domain.Product, error)
	FindByName(ctx context.Context, name string) (domain.Product, error)
	Update(ctx context.Context, tx Tx, id uuid.UUID, product domain.Product) (domain.Product, error)
	UpdateStock(ctx context.Context, tx Tx, productID uuid.UUID, delta int) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductService struct {
	pool       ConnPool
	repository ProductRepository
}

func NewProductService(pool ConnPool, repo ProductRepository) *ProductService {
	return &ProductService{
		pool:       pool,
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

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		return dto.ProductResponse{}, err
	}

	product := domain.Product{
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

	product := domain.Product{
		Name:  request.Name,
		Price: *request.Price,
		Stock: *request.Stock,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.ProductResponse{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := s.repository.FindByIDForUpdate(ctx, tx, id); err != nil {
		return dto.ProductResponse{}, err
	}

	product, err = s.repository.Update(
		ctx,
		tx,
		id,
		product,
	)

	if err != nil {
		return dto.ProductResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.NewProductResponse(product), nil
}

func (s *ProductService) Reserve(
	ctx context.Context,
	id uuid.UUID,
	quantity int,
) (dto.ProductResponse, error) {

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.ProductResponse{}, err
	}
	defer tx.Rollback(ctx)

	product, err := s.repository.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	if err := product.Reserve(quantity); err != nil {
		return dto.ProductResponse{}, err
	}

	product, err = s.repository.Update(ctx, tx, id, product)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.NewProductResponse(product), nil
}

func (s *ProductService) Release(
	ctx context.Context,
	id uuid.UUID,
	quantity int,
) (dto.ProductResponse, error) {

	if quantity <= 0 {
		return dto.ProductResponse{}, custom_errors.ErrOrderItemQuantityInvalid
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.ProductResponse{}, err
	}
	defer tx.Rollback(ctx)

	product, err := s.repository.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	product.Release(quantity)

	product, err = s.repository.Update(ctx, tx, id, product)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
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
