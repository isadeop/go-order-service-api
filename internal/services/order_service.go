package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	Create(ctx context.Context, tx pgx.Tx, order model.Order) (model.Order, error)
	FindByID(ctx context.Context, id uuid.UUID) (model.Order, error)
	FindAll(ctx context.Context, limit int, offset int) ([]model.Order, error)
	UpdateTotal(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, total float64) error
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status model.OrderStatus) (model.Order, error)
}

type OrderItemRepository interface {
	Create(ctx context.Context, tx pgx.Tx, item model.OrderItem) (model.OrderItem, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error)
}

type OrderService struct {
	pool        *pgxpool.Pool
	orderRepo   OrderRepository
	itemRepo    OrderItemRepository
	productRepo ProductRepository
	clientRepo  ClientRepository
}

func NewOrderService(pool *pgxpool.Pool,
	orderRepo OrderRepository,
	itemRepo OrderItemRepository,
	productRepo ProductRepository,
	clientRepo ClientRepository) *OrderService {

	return &OrderService{
		pool:        pool,
		orderRepo:   orderRepo,
		itemRepo:    itemRepo,
		productRepo: productRepo,
		clientRepo:  clientRepo,
	}
}

func (s *OrderService) Create(ctx context.Context, request dto.CreateOrderRequest) (dto.OrderResponse, error) {

	if request.ClientID == uuid.Nil {
		return dto.OrderResponse{}, custom_errors.ErrOrderClientRequired
	}

	if len(request.Items) == 0 {
		return dto.OrderResponse{}, custom_errors.ErrOrderItemsRequired
	}

	_, err := s.clientRepo.FindByID(ctx, request.ClientID)

	if errors.Is(err, custom_errors.ErrClientNotFound) {
		return dto.OrderResponse{}, custom_errors.ErrOrderClientNotFound
	}

	if err != nil {
		return dto.OrderResponse{}, err
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return dto.OrderResponse{}, err
	}

	defer tx.Rollback(ctx)

	order := model.Order{
		ClientID: request.ClientID,
		Status:   model.OrderStatusPending,
		Total:    0,
	}

	order, err = s.orderRepo.Create(ctx, tx, order)

	if err != nil {
		return dto.OrderResponse{}, err
	}

	var total float64

	itemsResponse := make([]dto.OrderItemResponse, 0)

	for _, itemRequest := range request.Items {
		if itemRequest.ProductID == uuid.Nil {
			return dto.OrderResponse{},
				custom_errors.ErrOrderItemProductRequired
		}

		if itemRequest.Quantity == nil {
			return dto.OrderResponse{},
				custom_errors.ErrOrderItemQuantityRequired
		}

		if *itemRequest.Quantity <= 0 {
			return dto.OrderResponse{},
				custom_errors.ErrOrderItemQuantityInvalid
		}

		product, err := s.productRepo.FindByID(ctx, itemRequest.ProductID)

		if errors.Is(err, custom_errors.ErrProductNotFound) {
			return dto.OrderResponse{},
				custom_errors.ErrOrderProductNotFound
		}

		if err != nil {
			return dto.OrderResponse{}, err
		}

		if product.Stock < *itemRequest.Quantity {
			return dto.OrderResponse{},
				custom_errors.ErrInsufficientStock
		}

		itemTotal := product.Price * float64(*itemRequest.Quantity)

		item := model.OrderItem{
			OrderID:   order.ID,
			ProductID: product.ID,
			Quantity:  *itemRequest.Quantity,
			Price:     product.Price,
		}

		item, err = s.itemRepo.Create(ctx, tx, item)

		if err != nil {
			return dto.OrderResponse{}, err
		}

		err = s.productRepo.UpdateStock(ctx, tx, product.ID, product.Stock-*itemRequest.Quantity)

		if err != nil {
			return dto.OrderResponse{}, err
		}

		total += itemTotal
		itemsResponse = append(itemsResponse, dto.NewOrderItemResponse(item))
	}

	err = s.orderRepo.UpdateTotal(ctx, tx, order.ID, total)

	if err != nil {
		return dto.OrderResponse{}, err
	}

	order.Total = total

	if err = tx.Commit(ctx); err != nil {
		return dto.OrderResponse{},
			fmt.Errorf("commit order: %w", err)
	}

	response := dto.NewOrderResponse(order)
	response.Items = itemsResponse

	return response, nil
}

func (s *OrderService) FindByID(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {

	order, err := s.orderRepo.FindByID(ctx, id)

	if err != nil {
		return dto.OrderResponse{}, err
	}

	items, err := s.itemRepo.FindByOrderID(ctx, order.ID)

	if err != nil {
		return dto.OrderResponse{}, err
	}

	response := dto.NewOrderResponse(order)

	response.Items = make(
		[]dto.OrderItemResponse,
		0,
		len(items),
	)

	for _, item := range items {
		response.Items = append(
			response.Items,
			dto.NewOrderItemResponse(item),
		)
	}

	return response, nil
}

func (s *OrderService) FindAll(ctx context.Context, limit int, offset int) ([]dto.OrderResponse, error) {

	orders, err := s.orderRepo.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	response := make(
		[]dto.OrderResponse,
		0,
		len(orders),
	)

	for _, order := range orders {

		orderResponse := dto.NewOrderResponse(order)

		items, err := s.itemRepo.FindByOrderID(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		orderResponse.Items = make(
			[]dto.OrderItemResponse,
			0,
			len(items),
		)

		for _, item := range items {
			orderResponse.Items = append(
				orderResponse.Items,
				dto.NewOrderItemResponse(item),
			)
		}

		response = append(
			response,
			orderResponse,
		)
	}
	return response, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) (dto.OrderResponse, error) {

	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	switch order.Status {
	case model.OrderStatusPaid:
		return dto.OrderResponse{},
			custom_errors.ErrOrderAlreadyPaid
	case model.OrderStatusCanceled:
		return dto.OrderResponse{},
			custom_errors.ErrOrderAlreadyCanceled
	}

	if status != model.OrderStatusPaid && status != model.OrderStatusCanceled {
		return dto.OrderResponse{},
			custom_errors.ErrInvalidOrderStatus
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	defer tx.Rollback(ctx)

	if status == model.OrderStatusCanceled {

		items, err := s.itemRepo.FindByOrderID(ctx, order.ID)
		if err != nil {
			return dto.OrderResponse{}, err
		}

		for _, item := range items {
			product, err := s.productRepo.FindByID(ctx, item.ProductID)
			if err != nil {
				return dto.OrderResponse{}, err
			}

			newStock := product.Stock + item.Quantity

			err = s.productRepo.UpdateStock(ctx, tx, product.ID, newStock)
			if err != nil {
				return dto.OrderResponse{}, err
			}
		}
	}

	updatedOrder, err := s.orderRepo.UpdateStatus(ctx, tx, id, status)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return dto.OrderResponse{}, fmt.Errorf("commit update order status: %w", err)
	}

	return s.FindByID(ctx, updatedOrder.ID)
}
