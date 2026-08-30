package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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

type StockReservationRepository interface {
	FindForUpdate(ctx context.Context, tx Tx, sagaID uuid.UUID, productID uuid.UUID) (domain.StockReservation, error)
	Create(ctx context.Context, tx Tx, reservation domain.StockReservation) error
	UpdateStatus(ctx context.Context, tx Tx, sagaID uuid.UUID, productID uuid.UUID, status domain.StockReservationStatus) error
	FindStaleReserved(ctx context.Context, olderThan time.Time) ([]domain.StockReservation, error)
}

type ProductService struct {
	pool         ConnPool
	repository   ProductRepository
	reservations StockReservationRepository
}

func NewProductService(pool ConnPool, repo ProductRepository, reservations StockReservationRepository) *ProductService {
	return &ProductService{
		pool:         pool,
		repository:   repo,
		reservations: reservations,
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

// Reserve decrementa o estoque de um produto em nome de uma saga
// (sagaID)
func (s *ProductService) Reserve(
	ctx context.Context,
	sagaID uuid.UUID,
	id uuid.UUID,
	quantity int,
) (dto.ProductResponse, error) {

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.ProductResponse{}, err
	}
	defer tx.Rollback(ctx)

	_, err = s.reservations.FindForUpdate(ctx, tx, sagaID, id)

	if err == nil {
		product, findErr := s.repository.FindByIDForUpdate(ctx, tx, id)
		if findErr != nil {
			return dto.ProductResponse{}, findErr
		}
		if err := tx.Commit(ctx); err != nil {
			return dto.ProductResponse{}, err
		}
		return dto.NewProductResponse(product), nil
	}

	if !errors.Is(err, custom_errors.ErrStockReservationNotFound) {
		return dto.ProductResponse{}, err
	}

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

	if err := s.reservations.Create(ctx, tx, domain.StockReservation{
		SagaID:    sagaID,
		ProductID: id,
		Quantity:  quantity,
		Status:    domain.StockReservationStatusReserved,
	}); err != nil {
		return dto.ProductResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.NewProductResponse(product), nil
}

// Release devolve ao estoque a quantidade reservada por uma saga (sagaID)
// para um produto. Só devolve efetivamente quando encontra uma reserva ainda com status
// RESERVED, e usa a quantidade registrada nela
func (s *ProductService) Release(
	ctx context.Context,
	sagaID uuid.UUID,
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

	reservation, err := s.reservations.FindForUpdate(ctx, tx, sagaID, id)

	if errors.Is(err, custom_errors.ErrStockReservationNotFound) {
		product, findErr := s.repository.FindByID(ctx, id)
		if findErr != nil {
			return dto.ProductResponse{}, findErr
		}
		if err := tx.Commit(ctx); err != nil {
			return dto.ProductResponse{}, err
		}
		return dto.NewProductResponse(product), nil
	}

	if err != nil {
		return dto.ProductResponse{}, err
	}

	if reservation.Status == domain.StockReservationStatusReleased {
		product, findErr := s.repository.FindByIDForUpdate(ctx, tx, id)
		if findErr != nil {
			return dto.ProductResponse{}, findErr
		}
		if err := tx.Commit(ctx); err != nil {
			return dto.ProductResponse{}, err
		}
		return dto.NewProductResponse(product), nil
	}

	product, err := s.repository.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	product.Release(reservation.Quantity)

	product, err = s.repository.Update(ctx, tx, id, product)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	if err := s.reservations.UpdateStatus(ctx, tx, sagaID, id, domain.StockReservationStatusReleased); err != nil {
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

// ReconcileStaleReservations libera reservas de estoque que ficaram
// paradas em RESERVED por mais tempo, e é chamado periodicamente por
// cmd/stock-service/main.go.
func (s *ProductService) ReconcileStaleReservations(ctx context.Context, staleAfter time.Duration) (releasedCount int, err error) {

	cutoff := time.Now().Add(-staleAfter)

	stale, err := s.reservations.FindStaleReserved(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("find stale reservations: %w", err)
	}

	for _, reservation := range stale {

		if _, releaseErr := s.Release(ctx, reservation.SagaID, reservation.ProductID, reservation.Quantity); releaseErr != nil {
			slog.Error("stock.reconciliation.release_failed",
				"operation", "ReconcileStaleReservations",
				"result", "error",
				"saga_id", reservation.SagaID.String(),
				"product_id", reservation.ProductID.String(),
				"quantity", reservation.Quantity,
				"err", releaseErr.Error(),
			)
			continue
		}

		slog.Warn("stock.reconciliation.released_orphan",
			"operation", "ReconcileStaleReservations",
			"result", "ok",
			"saga_id", reservation.SagaID.String(),
			"product_id", reservation.ProductID.String(),
			"quantity", reservation.Quantity,
			"reserved_since", reservation.CreatedAt,
		)

		releasedCount++
	}

	return releasedCount, nil
}
