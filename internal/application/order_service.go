package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/txport"
)

// Tx e ConnPool são aliases para o contrato compartilhado em txport
type Tx = txport.Tx
type ConnPool = txport.ConnPool

type OrderRepository interface {
	Create(ctx context.Context, tx Tx, order domain.Order) (domain.Order, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Order, error)
	FindByIDForUpdate(ctx context.Context, tx Tx, id uuid.UUID) (domain.Order, error)
	FindAll(ctx context.Context, limit int, offset int) ([]domain.Order, error)
	UpdateTotal(ctx context.Context, tx Tx, orderID uuid.UUID, total float64) error
	UpdateStatus(ctx context.Context, tx Tx, id uuid.UUID, status domain.OrderStatus) (domain.Order, error)
}

type OrderItemRepository interface {
	Create(ctx context.Context, tx Tx, item domain.OrderItem) (domain.OrderItem, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error)
}

// ProductStockGateway é a porta que o order-service usa para consultar e
// reservar/liberar estoque de produtos.
type ProductStockGateway interface {
	FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	Reserve(ctx context.Context, productID uuid.UUID, quantity int) error
	Release(ctx context.Context, productID uuid.UUID, quantity int) error
}

// reservedItem registra uma reserva de estoque já confirmada pelo
// stock-service durante Create.
type reservedItem struct {
	productID uuid.UUID
	quantity  int
}

type OrderService struct {
	pool         ConnPool
	orderRepo    OrderRepository
	itemRepo     OrderItemRepository
	productStock ProductStockGateway
	clientRepo   ClientRepository
}

func NewOrderService(pool ConnPool,
	orderRepo OrderRepository,
	itemRepo OrderItemRepository,
	productStock ProductStockGateway,
	clientRepo ClientRepository) *OrderService {

	return &OrderService{
		pool:         pool,
		orderRepo:    orderRepo,
		itemRepo:     itemRepo,
		productStock: productStock,
		clientRepo:   clientRepo,
	}
}

func (s *OrderService) compensateReservations(ctx context.Context, reserved []reservedItem) {
	for _, item := range reserved {
		if err := s.productStock.Release(ctx, item.productID, item.quantity); err != nil {
			slog.Error("order.create.compensation_failed",
				"operation", "Create",
				"result", "error",
				"product_id", item.productID.String(),
				"quantity", item.quantity,
				"err", err.Error(),
			)
		}
	}
}

func (s *OrderService) Create(ctx context.Context, request dto.CreateOrderRequest) (dto.OrderResponse, error) {

	if request.ClientID == uuid.Nil {
		return dto.OrderResponse{}, custom_errors.ErrOrderClientRequired
	}

	if len(request.Items) == 0 {
		return dto.OrderResponse{}, custom_errors.ErrOrderItemsRequired
	}

	client, err := s.clientRepo.FindByID(ctx, request.ClientID)

	if errors.Is(err, custom_errors.ErrClientNotFound) {
		return dto.OrderResponse{}, custom_errors.ErrOrderClientNotFound
	}

	if err != nil {
		return dto.OrderResponse{}, err
	}

	quantities := make(map[uuid.UUID]int)
	productOrder := make([]uuid.UUID, 0, len(request.Items))

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

		if _, alreadySeen := quantities[itemRequest.ProductID]; !alreadySeen {
			productOrder = append(productOrder, itemRequest.ProductID)
		}

		quantities[itemRequest.ProductID] += *itemRequest.Quantity
	}

	// Busca os dados de cada produto (preço, nome) no stock-service.
	products := make(map[uuid.UUID]domain.Product, len(productOrder))

	for _, productID := range productOrder {

		product, err := s.productStock.FindByID(ctx, productID)

		if errors.Is(err, custom_errors.ErrProductNotFound) {
			return dto.OrderResponse{},
				custom_errors.ErrOrderProductNotFound
		}

		if err != nil {
			return dto.OrderResponse{}, err
		}

		products[productID] = product
	}

	reserved := make([]reservedItem, 0, len(productOrder))

	for _, productID := range productOrder {

		quantity := quantities[productID]

		if err := s.productStock.Reserve(ctx, productID, quantity); err != nil {
			s.compensateReservations(ctx, reserved)
			return dto.OrderResponse{}, err
		}

		reserved = append(reserved, reservedItem{productID: productID, quantity: quantity})
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		s.compensateReservations(ctx, reserved)
		return dto.OrderResponse{}, err
	}

	defer tx.Rollback(ctx)

	order := domain.Order{
		ClientID: request.ClientID,
		Status:   domain.OrderStatusPending,
		Total:    0,
	}

	order, err = s.orderRepo.Create(ctx, tx, order)

	if err != nil {
		s.compensateReservations(ctx, reserved)
		return dto.OrderResponse{}, err
	}

	var total float64

	itemsResponse := make([]dto.OrderItemResponse, 0, len(request.Items))

	for _, itemRequest := range request.Items {

		product := products[itemRequest.ProductID]

		itemTotal := product.Price * float64(*itemRequest.Quantity)

		item := domain.OrderItem{
			OrderID:   order.ID,
			ProductID: product.ID,
			Quantity:  *itemRequest.Quantity,
			Price:     product.Price,
		}

		item, err = s.itemRepo.Create(ctx, tx, item)

		if err != nil {
			s.compensateReservations(ctx, reserved)
			return dto.OrderResponse{}, err
		}

		total += itemTotal
		itemsResponse = append(itemsResponse, dto.NewOrderItemResponse(item, product.Name))
	}

	err = s.orderRepo.UpdateTotal(ctx, tx, order.ID, total)

	if err != nil {
		s.compensateReservations(ctx, reserved)
		return dto.OrderResponse{}, err
	}

	order.Total = total

	if err = tx.Commit(ctx); err != nil {
		s.compensateReservations(ctx, reserved)
		return dto.OrderResponse{},
			fmt.Errorf("commit order: %w", err)
	}

	response := dto.NewOrderResponse(order, client.Name)
	response.Items = itemsResponse

	return response, nil
}

func (s *OrderService) FindByID(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {

	order, err := s.orderRepo.FindByID(ctx, id)

	if err != nil {
		return dto.OrderResponse{}, err
	}

	client, err := s.clientRepo.FindByID(ctx, order.ClientID)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	items, err := s.itemRepo.FindByOrderID(ctx, order.ID)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	response := dto.NewOrderResponse(order, client.Name)

	response.Items = make(
		[]dto.OrderItemResponse,
		0,
		len(items),
	)

	for _, item := range items {

		product, err := s.productStock.FindByID(ctx, item.ProductID)
		if err != nil {
			return dto.OrderResponse{}, err
		}

		response.Items = append(
			response.Items,
			dto.NewOrderItemResponse(item, product.Name),
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
		client, err := s.clientRepo.FindByID(ctx, order.ClientID)
		if err != nil {
			return nil, err
		}

		orderResponse := dto.NewOrderResponse(order, client.Name)

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
			product, err := s.productStock.FindByID(ctx, item.ProductID)
			if err != nil {
				return nil, err
			}
			orderResponse.Items = append(
				orderResponse.Items,
				dto.NewOrderItemResponse(item, product.Name),
			)
		}

		response = append(
			response,
			orderResponse,
		)
	}
	return response, nil
}

// refundItemsStock devolve ao stock-service, via HTTP, o estoque de todos
// os itens do pedido. Chamada antes do commit da transação local que muda
// o status do pedido para CANCELED: se a liberação falhar, o cancelamento
// inteiro é abortado e o pedido continua no status anterior — falha segura
// (nenhum estado fica inconsistente). O gap conhecido é o caminho inverso
// (liberação funciona, mas o commit local falha logo em seguida) — tratado
// de fato só com idempotência na Saga (Etapa 10).
func (s *OrderService) refundItemsStock(ctx context.Context, orderID uuid.UUID) error {

	items, err := s.itemRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := s.productStock.Release(ctx, item.ProductID, item.Quantity); err != nil {
			return err
		}
	}

	return nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (dto.OrderResponse, error) {

	if status != domain.OrderStatusPaid && status != domain.OrderStatusCanceled {
		return dto.OrderResponse{},
			custom_errors.ErrInvalidOrderStatus
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	defer tx.Rollback(ctx)

	order, err := s.orderRepo.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	if err := order.ChangeStatus(status); err != nil {
		return dto.OrderResponse{}, err
	}

	if status == domain.OrderStatusCanceled {
		if err := s.refundItemsStock(ctx, order.ID); err != nil {
			return dto.OrderResponse{}, err
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

func (s *OrderService) Pay(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	defer tx.Rollback(ctx)

	order, err := s.orderRepo.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	if err := order.Pay(); err != nil {
		return dto.OrderResponse{}, err
	}

	order, err = s.orderRepo.UpdateStatus(ctx, tx, id, domain.OrderStatusPaid)
	if err != nil {
		return dto.OrderResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.OrderResponse{}, err
	}

	return s.FindByID(ctx, order.ID)
}

func (s *OrderService) Cancel(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.OrderResponse{}, err
	}
	defer tx.Rollback(ctx)

	order, err := s.orderRepo.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return dto.OrderResponse{}, err
	}

	if err := order.Cancel(); err != nil {
		return dto.OrderResponse{}, err
	}

	if err := s.refundItemsStock(ctx, order.ID); err != nil {
		return dto.OrderResponse{}, err
	}

	order, err = s.orderRepo.UpdateStatus(ctx, tx, id, domain.OrderStatusCanceled)
	if err != nil {
		return dto.OrderResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.OrderResponse{}, err
	}
	return s.FindByID(ctx, order.ID)
}
