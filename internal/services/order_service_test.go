package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
)

func intPtr(v int) *int {
	return &v
}

type fakeOrderRepository struct {
	byID            map[uuid.UUID]model.Order
	order           []uuid.UUID
	findAllErr      error
	updateStatusErr error
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{byID: make(map[uuid.UUID]model.Order)}
}

func (f *fakeOrderRepository) Create(ctx context.Context, tx pgx.Tx, order model.Order) (model.Order, error) {
	order.ID = uuid.New()
	f.byID[order.ID] = order
	f.order = append(f.order, order.ID)
	return order, nil
}

func (f *fakeOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Order, error) {
	order, ok := f.byID[id]
	if !ok {
		return model.Order{}, custom_errors.ErrOrderNotFound
	}
	return order, nil
}

func (f *fakeOrderRepository) FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (model.Order, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeOrderRepository) FindAll(ctx context.Context, limit, offset int) ([]model.Order, error) {
	if f.findAllErr != nil {
		return nil, f.findAllErr
	}

	orders := make([]model.Order, 0, len(f.order))
	for _, id := range f.order {
		orders = append(orders, f.byID[id])
	}

	if offset >= len(orders) {
		return []model.Order{}, nil
	}

	end := offset + limit
	if end > len(orders) {
		end = len(orders)
	}

	return orders[offset:end], nil
}

func (f *fakeOrderRepository) UpdateTotal(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, total float64) error {
	order, ok := f.byID[orderID]
	if !ok {
		return custom_errors.ErrOrderNotFound
	}
	order.Total = total
	f.byID[orderID] = order
	return nil
}

func (f *fakeOrderRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status model.OrderStatus) (model.Order, error) {
	if f.updateStatusErr != nil {
		return model.Order{}, f.updateStatusErr
	}
	order, ok := f.byID[id]
	if !ok {
		return model.Order{}, custom_errors.ErrOrderNotFound
	}
	order.Status = status
	f.byID[id] = order
	return order, nil
}

type fakeOrderItemRepository struct {
	byOrderID map[uuid.UUID][]model.OrderItem
}

func newFakeOrderItemRepository() *fakeOrderItemRepository {
	return &fakeOrderItemRepository{byOrderID: make(map[uuid.UUID][]model.OrderItem)}
}

func (f *fakeOrderItemRepository) Create(ctx context.Context, tx pgx.Tx, item model.OrderItem) (model.OrderItem, error) {
	item.ID = uuid.New()
	f.byOrderID[item.OrderID] = append(f.byOrderID[item.OrderID], item)
	return item, nil
}

func (f *fakeOrderItemRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error) {
	return f.byOrderID[orderID], nil
}

type erroringClientRepository struct {
	err error
}

func (e *erroringClientRepository) Create(ctx context.Context, client model.Client) (model.Client, error) {
	return model.Client{}, e.err
}

func (e *erroringClientRepository) FindAll(ctx context.Context) ([]model.Client, error) {
	return nil, e.err
}

func (e *erroringClientRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Client, error) {
	return model.Client{}, e.err
}

func (e *erroringClientRepository) FindByEmail(ctx context.Context, email string) (model.Client, error) {
	return model.Client{}, e.err
}

// orderServiceFixture agrupa os fakes usados para montar um OrderService pronto para teste
type orderServiceFixture struct {
	pool        *fakeConnPool
	orderRepo   *fakeOrderRepository
	itemRepo    *fakeOrderItemRepository
	productRepo *fakeProductRepository
	clientRepo  *fakeClientRepository
	service     *OrderService
}

func newOrderServiceFixture() *orderServiceFixture {
	f := &orderServiceFixture{
		pool:        newFakeConnPool(),
		orderRepo:   newFakeOrderRepository(),
		itemRepo:    newFakeOrderItemRepository(),
		productRepo: newFakeProductRepository(),
		clientRepo:  newFakeClientRepository(),
	}
	f.service = NewOrderService(f.pool, f.orderRepo, f.itemRepo, f.productRepo, f.clientRepo)
	return f
}

func setupClientAndProduct(f *orderServiceFixture, stock int) (clientID, productID uuid.UUID) {
	client, _ := f.clientRepo.Create(context.Background(), model.Client{
		Name:  "Cliente Teste",
		Email: uuid.NewString() + "@teste.com",
	})

	product, _ := f.productRepo.Create(context.Background(), model.Product{
		Name:  uuid.NewString(),
		Price: 10,
		Stock: stock,
	})

	return client.ID, product.ID
}

func seedOrder(f *orderServiceFixture, clientID uuid.UUID, status model.OrderStatus, total float64) uuid.UUID {
	order, _ := f.orderRepo.Create(context.Background(), nil, model.Order{
		ClientID: clientID,
		Status:   status,
		Total:    total,
	})
	return order.ID
}

func TestOrderService_Create_ClienteObrigatorio(t *testing.T) {
	f := newOrderServiceFixture()

	request := dto.CreateOrderRequest{
		Items: []dto.CreateOrderItemRequest{{ProductID: uuid.New(), Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, custom_errors.ErrOrderClientRequired) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderClientRequired)
	}
}

func TestOrderService_Create_ItensObrigatorios(t *testing.T) {
	f := newOrderServiceFixture()

	request := dto.CreateOrderRequest{ClientID: uuid.New()}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, custom_errors.ErrOrderItemsRequired) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderItemsRequired)
	}
}

func TestOrderService_Create_ClienteInexistente(t *testing.T) {
	f := newOrderServiceFixture()

	request := dto.CreateOrderRequest{
		ClientID: uuid.New(), // não cadastrado no fake
		Items:    []dto.CreateOrderItemRequest{{ProductID: uuid.New(), Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, custom_errors.ErrOrderClientNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderClientNotFound)
	}
}

func TestOrderService_Create_ErroInesperadoAoBuscarCliente(t *testing.T) {
	infraErr := errors.New("timeout de conexão")
	f := newOrderServiceFixture()
	f.service = NewOrderService(f.pool, f.orderRepo, f.itemRepo, f.productRepo, &erroringClientRepository{err: infraErr})

	request := dto.CreateOrderRequest{
		ClientID: uuid.New(),
		Items:    []dto.CreateOrderItemRequest{{ProductID: uuid.New(), Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, infraErr) {
		t.Errorf("esperava que o erro de infraestrutura fosse propagado, obteve: %v", err)
	}
}

func TestOrderService_Create_ValidacoesDeItem(t *testing.T) {
	tests := []struct {
		name    string
		stock   int
		item    func(productID uuid.UUID) dto.CreateOrderItemRequest
		wantErr error
	}{
		{
			name:  "produto obrigatório",
			stock: 10,
			item: func(uuid.UUID) dto.CreateOrderItemRequest {
				return dto.CreateOrderItemRequest{Quantity: intPtr(1)}
			},
			wantErr: custom_errors.ErrOrderItemProductRequired,
		},
		{
			name:  "quantidade obrigatória",
			stock: 10,
			item: func(productID uuid.UUID) dto.CreateOrderItemRequest {
				return dto.CreateOrderItemRequest{ProductID: productID}
			},
			wantErr: custom_errors.ErrOrderItemQuantityRequired,
		},
		{
			name:  "quantidade inválida",
			stock: 10,
			item: func(productID uuid.UUID) dto.CreateOrderItemRequest {
				return dto.CreateOrderItemRequest{ProductID: productID, Quantity: intPtr(0)}
			},
			wantErr: custom_errors.ErrOrderItemQuantityInvalid,
		},
		{
			name:  "produto inexistente",
			stock: 10,
			item: func(uuid.UUID) dto.CreateOrderItemRequest {
				return dto.CreateOrderItemRequest{ProductID: uuid.New(), Quantity: intPtr(1)}
			},
			wantErr: custom_errors.ErrOrderProductNotFound,
		},
		{
			name:  "estoque insuficiente",
			stock: 5,
			item: func(productID uuid.UUID) dto.CreateOrderItemRequest {
				return dto.CreateOrderItemRequest{ProductID: productID, Quantity: intPtr(6)}
			},
			wantErr: custom_errors.ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOrderServiceFixture()
			clientID, productID := setupClientAndProduct(f, tt.stock)

			request := dto.CreateOrderRequest{
				ClientID: clientID,
				Items:    []dto.CreateOrderItemRequest{tt.item(productID)},
			}

			_, err := f.service.Create(context.Background(), request)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderService_Create_ProdutosDuplicadosSomamQuantidade(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items: []dto.CreateOrderItemRequest{
			{ProductID: productID, Quantity: intPtr(4)},
			{ProductID: productID, Quantity: intPtr(4)},
		},
	}

	response, err := f.service.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("Create retornou erro inesperado: %v", err)
	}

	if response.Total != 80 {
		t.Errorf("total = %v, esperado 80 (10.0 * 8 unidades somadas)", response.Total)
	}
	if len(response.Items) != 2 {
		t.Errorf("esperava 2 itens na resposta, obteve %d", len(response.Items))
	}

	product, _ := f.productRepo.FindByID(context.Background(), productID)
	if product.Stock != 2 {
		t.Errorf("estoque restante = %d, esperado 2 (10 - 8)", product.Stock)
	}
}

func TestOrderService_Create_ProdutosDuplicadosExcedemEstoque(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 5)

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items: []dto.CreateOrderItemRequest{
			{ProductID: productID, Quantity: intPtr(3)},
			{ProductID: productID, Quantity: intPtr(3)},
		},
	}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, custom_errors.ErrInsufficientStock) {
		t.Errorf("erro = %v, esperado %v (soma das quantidades do mesmo produto no pedido deve ser validada contra o estoque)", err, custom_errors.ErrInsufficientStock)
	}
}

func TestOrderService_Create_HappyPath(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(3)}},
	}

	response, err := f.service.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("Create retornou erro inesperado: %v", err)
	}

	if response.Status != model.OrderStatusPending {
		t.Errorf("status = %v, esperado %v", response.Status, model.OrderStatusPending)
	}
	if response.Total != 30 {
		t.Errorf("total = %v, esperado 30", response.Total)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}

	product, _ := f.productRepo.FindByID(context.Background(), productID)
	if product.Stock != 7 {
		t.Errorf("estoque restante = %d, esperado 7", product.Stock)
	}
}

func TestOrderService_Create_RollbackQuandoAtualizarEstoqueFalha(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	f.productRepo.updateStockErr = errors.New("falha ao atualizar estoque")

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if err == nil {
		t.Fatal("esperava erro ao atualizar estoque")
	}
	if f.pool.tx.committed {
		t.Error("transação não deveria ter sido commitada quando UpdateStock falha")
	}
	if !f.pool.tx.rolledBack {
		t.Error("esperava rollback da transação quando UpdateStock falha")
	}
}

func TestOrderService_Create_ErroAoCommitarEhPropagado(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	commitErr := errors.New("falha de rede ao commitar")
	f.pool.tx.commitErr = commitErr

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, commitErr) {
		t.Errorf("esperava que o erro de commit fosse propagado, obteve: %v", err)
	}
}

func TestOrderService_Pay_HappyPath(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, _ := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 50)

	response, err := f.service.Pay(context.Background(), orderID)
	if err != nil {
		t.Fatalf("Pay retornou erro inesperado: %v", err)
	}
	if response.Status != model.OrderStatusPaid {
		t.Errorf("status = %v, esperado %v", response.Status, model.OrderStatusPaid)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}
}

func TestOrderService_Pay_PedidoInexistente(t *testing.T) {
	f := newOrderServiceFixture()

	_, err := f.service.Pay(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrOrderNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderNotFound)
	}
}

func TestOrderService_Pay_RegrasDeStatus(t *testing.T) {
	tests := []struct {
		name       string
		seedStatus model.OrderStatus
		wantErr    error
	}{
		{"pedido já pago", model.OrderStatusPaid, custom_errors.ErrOrderAlreadyPaid},
		{"pedido cancelado", model.OrderStatusCanceled, custom_errors.ErrOrderCannotChangeStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOrderServiceFixture()
			clientID, _ := setupClientAndProduct(f, 10)
			orderID := seedOrder(f, clientID, tt.seedStatus, 50)

			_, err := f.service.Pay(context.Background(), orderID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderService_Pay_RollbackQuandoAtualizarStatusFalha(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, _ := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 50)
	f.orderRepo.updateStatusErr = errors.New("falha ao atualizar status do pedido")

	_, err := f.service.Pay(context.Background(), orderID)

	if err == nil {
		t.Fatal("esperava erro ao atualizar o status do pedido")
	}
	if f.pool.tx.committed {
		t.Error("transação não deveria ter sido commitada quando UpdateStatus falha")
	}
	if !f.pool.tx.rolledBack {
		t.Error("esperava rollback da transação quando UpdateStatus falha")
	}
}

func TestOrderService_Cancel_HappyPathEstornaEstoque(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	// Simula que 3 unidades já haviam sido debitadas na criação do pedido.
	if err := f.productRepo.UpdateStock(context.Background(), nil, productID, -3); err != nil {
		t.Fatalf("setup do estoque falhou: %v", err)
	}
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 30)
	if _, err := f.itemRepo.Create(context.Background(), nil, model.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 3, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}

	response, err := f.service.Cancel(context.Background(), orderID)
	if err != nil {
		t.Fatalf("Cancel retornou erro inesperado: %v", err)
	}
	if response.Status != model.OrderStatusCanceled {
		t.Errorf("status = %v, esperado %v", response.Status, model.OrderStatusCanceled)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}

	product, _ := f.productRepo.FindByID(context.Background(), productID)
	if product.Stock != 10 {
		t.Errorf("estoque após estorno = %d, esperado 10 (estoque original restaurado)", product.Stock)
	}
}

func TestOrderService_Cancel_RegrasDeStatus(t *testing.T) {
	tests := []struct {
		name       string
		seedStatus model.OrderStatus
		wantErr    error
	}{
		{"pedido já cancelado", model.OrderStatusCanceled, custom_errors.ErrOrderAlreadyCanceled},
		{"pedido já pago", model.OrderStatusPaid, custom_errors.ErrOrderCannotChangeStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOrderServiceFixture()
			clientID, _ := setupClientAndProduct(f, 10)
			orderID := seedOrder(f, clientID, tt.seedStatus, 30)

			_, err := f.service.Cancel(context.Background(), orderID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderService_Cancel_PedidoInexistente(t *testing.T) {
	f := newOrderServiceFixture()

	_, err := f.service.Cancel(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrOrderNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderNotFound)
	}
}

func TestOrderService_Cancel_RollbackQuandoEstornoFalha(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 30)
	if _, err := f.itemRepo.Create(context.Background(), nil, model.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 3, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}
	f.productRepo.updateStockErr = errors.New("falha ao estornar estoque")

	_, err := f.service.Cancel(context.Background(), orderID)

	if err == nil {
		t.Fatal("esperava erro no estorno de estoque")
	}
	if f.pool.tx.committed {
		t.Error("transação não deveria ter sido commitada quando o estorno de estoque falha")
	}
	if !f.pool.tx.rolledBack {
		t.Error("esperava rollback quando o estorno de estoque falha")
	}

	order, _ := f.orderRepo.FindByID(context.Background(), orderID)
	if order.Status != model.OrderStatusPending {
		t.Errorf("status do pedido não deveria ter mudado após rollback, obteve %v", order.Status)
	}
}

func TestOrderService_UpdateStatus_RegrasInvalidas(t *testing.T) {
	tests := []struct {
		name         string
		seedStatus   model.OrderStatus
		targetStatus model.OrderStatus
		wantErr      error
	}{
		{"status alvo inválido", model.OrderStatusPending, model.OrderStatusPending, custom_errors.ErrInvalidOrderStatus},
		{"pedido já pago", model.OrderStatusPaid, model.OrderStatusCanceled, custom_errors.ErrOrderAlreadyPaid},
		{"pedido já cancelado", model.OrderStatusCanceled, model.OrderStatusPaid, custom_errors.ErrOrderAlreadyCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOrderServiceFixture()
			clientID, _ := setupClientAndProduct(f, 10)
			orderID := seedOrder(f, clientID, tt.seedStatus, 30)

			_, err := f.service.UpdateStatus(context.Background(), orderID, tt.targetStatus)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderService_UpdateStatus_ParaPagoHappyPath(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, _ := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 30)

	response, err := f.service.UpdateStatus(context.Background(), orderID, model.OrderStatusPaid)

	if err != nil {
		t.Fatalf("UpdateStatus retornou erro inesperado: %v", err)
	}
	if response.Status != model.OrderStatusPaid {
		t.Errorf("status = %v, esperado %v", response.Status, model.OrderStatusPaid)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}
}

func TestOrderService_UpdateStatus_ParaCanceladoEstornaEstoque(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	if err := f.productRepo.UpdateStock(context.Background(), nil, productID, -4); err != nil {
		t.Fatalf("setup do estoque falhou: %v", err)
	}
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 40)
	if _, err := f.itemRepo.Create(context.Background(), nil, model.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 4, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}

	_, err := f.service.UpdateStatus(context.Background(), orderID, model.OrderStatusCanceled)
	if err != nil {
		t.Fatalf("UpdateStatus retornou erro inesperado: %v", err)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}

	product, _ := f.productRepo.FindByID(context.Background(), productID)
	if product.Stock != 10 {
		t.Errorf("estoque após estorno = %d, esperado 10", product.Stock)
	}
}

func TestOrderService_FindByID_HappyPath(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, model.OrderStatusPending, 30)
	if _, err := f.itemRepo.Create(context.Background(), nil, model.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 3, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}

	response, err := f.service.FindByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if response.ClientName == "" {
		t.Error("esperava que o nome do cliente fosse preenchido")
	}
	if len(response.Items) != 1 || response.Items[0].ProductName == "" {
		t.Errorf("esperava 1 item com nome de produto preenchido, obteve %+v", response.Items)
	}
}

func TestOrderService_FindByID_PedidoInexistente(t *testing.T) {
	f := newOrderServiceFixture()

	_, err := f.service.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrOrderNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrOrderNotFound)
	}
}

func TestOrderService_FindAll_AgregaClienteEItensDeVariosPedidos(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	firstID := seedOrder(f, clientID, model.OrderStatusPending, 10)
	secondID := seedOrder(f, clientID, model.OrderStatusPaid, 20)
	if _, err := f.itemRepo.Create(context.Background(), nil, model.OrderItem{OrderID: firstID, ProductID: productID, Quantity: 1, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}
	if _, err := f.itemRepo.Create(context.Background(), nil, model.OrderItem{OrderID: secondID, ProductID: productID, Quantity: 2, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}

	responses, err := f.service.FindAll(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("esperava 2 pedidos, obteve %d", len(responses))
	}
	for _, response := range responses {
		if response.ClientName == "" {
			t.Error("esperava nome do cliente preenchido em todos os pedidos")
		}
		if len(response.Items) != 1 {
			t.Errorf("esperava 1 item por pedido, obteve %d", len(response.Items))
		}
	}
}

func TestOrderService_FindAll_PropagaErroDoRepository(t *testing.T) {
	f := newOrderServiceFixture()
	infraErr := errors.New("falha de conexão")
	f.orderRepo.findAllErr = infraErr

	_, err := f.service.FindAll(context.Background(), 10, 0)

	if !errors.Is(err, infraErr) {
		t.Errorf("esperava que o erro de infraestrutura fosse propagado, obteve: %v", err)
	}
}
