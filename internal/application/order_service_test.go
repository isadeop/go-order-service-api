package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

func intPtr(v int) *int {
	return &v
}

type fakeOrderRepository struct {
	byID            map[uuid.UUID]domain.Order
	order           []uuid.UUID
	findAllErr      error
	updateStatusErr error
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{byID: make(map[uuid.UUID]domain.Order)}
}

func (f *fakeOrderRepository) Create(ctx context.Context, tx Tx, order domain.Order) (domain.Order, error) {
	order.ID = uuid.New()
	f.byID[order.ID] = order
	f.order = append(f.order, order.ID)
	return order, nil
}

func (f *fakeOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	order, ok := f.byID[id]
	if !ok {
		return domain.Order{}, custom_errors.ErrOrderNotFound
	}
	return order, nil
}

func (f *fakeOrderRepository) FindByIDForUpdate(ctx context.Context, tx Tx, id uuid.UUID) (domain.Order, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeOrderRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Order, error) {
	if f.findAllErr != nil {
		return nil, f.findAllErr
	}

	orders := make([]domain.Order, 0, len(f.order))
	for _, id := range f.order {
		orders = append(orders, f.byID[id])
	}

	if offset >= len(orders) {
		return []domain.Order{}, nil
	}

	end := offset + limit
	if end > len(orders) {
		end = len(orders)
	}

	return orders[offset:end], nil
}

func (f *fakeOrderRepository) UpdateTotal(ctx context.Context, tx Tx, orderID uuid.UUID, total float64) error {
	order, ok := f.byID[orderID]
	if !ok {
		return custom_errors.ErrOrderNotFound
	}
	order.Total = total
	f.byID[orderID] = order
	return nil
}

func (f *fakeOrderRepository) UpdateStatus(ctx context.Context, tx Tx, id uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	if f.updateStatusErr != nil {
		return domain.Order{}, f.updateStatusErr
	}
	order, ok := f.byID[id]
	if !ok {
		return domain.Order{}, custom_errors.ErrOrderNotFound
	}
	order.Status = status
	f.byID[id] = order
	return order, nil
}

type fakeOrderItemRepository struct {
	byOrderID map[uuid.UUID][]domain.OrderItem
}

func newFakeOrderItemRepository() *fakeOrderItemRepository {
	return &fakeOrderItemRepository{byOrderID: make(map[uuid.UUID][]domain.OrderItem)}
}

func (f *fakeOrderItemRepository) Create(ctx context.Context, tx Tx, item domain.OrderItem) (domain.OrderItem, error) {
	item.ID = uuid.New()
	f.byOrderID[item.OrderID] = append(f.byOrderID[item.OrderID], item)
	return item, nil
}

func (f *fakeOrderItemRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	return f.byOrderID[orderID], nil
}

type erroringClientRepository struct {
	err error
}

func (e *erroringClientRepository) Create(ctx context.Context, client domain.Client) (domain.Client, error) {
	return domain.Client{}, e.err
}

func (e *erroringClientRepository) FindAll(ctx context.Context) ([]domain.Client, error) {
	return nil, e.err
}

func (e *erroringClientRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	return domain.Client{}, e.err
}

func (e *erroringClientRepository) FindByEmail(ctx context.Context, email string) (domain.Client, error) {
	return domain.Client{}, e.err
}

type gatewayCall struct {
	sagaID    uuid.UUID
	productID uuid.UUID
	quantity  int
}

type fakeProductStockGateway struct {
	byID map[uuid.UUID]domain.Product

	reserveErr error
	releaseErr error

	reserveCalls []gatewayCall
	releaseCalls []gatewayCall
}

func newFakeProductStockGateway() *fakeProductStockGateway {
	return &fakeProductStockGateway{byID: make(map[uuid.UUID]domain.Product)}
}

// Create simula o produto já existindo no stock-service.
func (f *fakeProductStockGateway) Create(ctx context.Context, product domain.Product) (domain.Product, error) {
	product.ID = uuid.New()
	f.byID[product.ID] = product
	return product, nil
}

func (f *fakeProductStockGateway) FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	product, ok := f.byID[id]
	if !ok {
		return domain.Product{}, custom_errors.ErrProductNotFound
	}
	return product, nil
}

func (f *fakeProductStockGateway) Reserve(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	f.reserveCalls = append(f.reserveCalls, gatewayCall{sagaID: sagaID, productID: productID, quantity: quantity})

	if f.reserveErr != nil {
		return f.reserveErr
	}

	product, ok := f.byID[productID]
	if !ok {
		return custom_errors.ErrProductNotFound
	}

	if err := product.Reserve(quantity); err != nil {
		return err
	}

	f.byID[productID] = product
	return nil
}

func (f *fakeProductStockGateway) Release(ctx context.Context, sagaID uuid.UUID, productID uuid.UUID, quantity int) error {
	f.releaseCalls = append(f.releaseCalls, gatewayCall{sagaID: sagaID, productID: productID, quantity: quantity})

	if f.releaseErr != nil {
		return f.releaseErr
	}

	product, ok := f.byID[productID]
	if !ok {
		return custom_errors.ErrProductNotFound
	}

	product.Release(quantity)
	f.byID[productID] = product
	return nil
}

// orderServiceFixture agrupa os fakes usados para montar um OrderService pronto para teste
type orderServiceFixture struct {
	pool         *fakeConnPool
	orderRepo    *fakeOrderRepository
	itemRepo     *fakeOrderItemRepository
	productStock *fakeProductStockGateway
	clientRepo   *fakeClientRepository
	service      *OrderService
}

func newOrderServiceFixture() *orderServiceFixture {
	f := &orderServiceFixture{
		pool:         newFakeConnPool(),
		orderRepo:    newFakeOrderRepository(),
		itemRepo:     newFakeOrderItemRepository(),
		productStock: newFakeProductStockGateway(),
		clientRepo:   newFakeClientRepository(),
	}
	f.service = NewOrderService(f.pool, f.orderRepo, f.itemRepo, f.productStock, f.clientRepo)
	return f
}

func setupClientAndProduct(f *orderServiceFixture, stock int) (clientID, productID uuid.UUID) {
	client, _ := f.clientRepo.Create(context.Background(), domain.Client{
		Name:  "Cliente Teste",
		Email: uuid.NewString() + "@teste.com",
	})

	product, _ := f.productStock.Create(context.Background(), domain.Product{
		Name:  uuid.NewString(),
		Price: 10,
		Stock: stock,
	})

	return client.ID, product.ID
}

func seedOrder(f *orderServiceFixture, clientID uuid.UUID, status domain.OrderStatus, total float64) uuid.UUID {
	order, _ := f.orderRepo.Create(context.Background(), nil, domain.Order{
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
	f.service = NewOrderService(f.pool, f.orderRepo, f.itemRepo, f.productStock, &erroringClientRepository{err: infraErr})

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

	// Mesmo com 2 itens no pedido para o mesmo produto, a reserva no
	// stock-service deve acontecer uma única vez, com a quantidade somada.
	if len(f.productStock.reserveCalls) != 1 {
		t.Fatalf("esperava 1 chamada de Reserve (quantidade agregada), obteve %d", len(f.productStock.reserveCalls))
	}
	if f.productStock.reserveCalls[0].quantity != 8 {
		t.Errorf("quantidade reservada = %d, esperado 8", f.productStock.reserveCalls[0].quantity)
	}

	product, _ := f.productStock.FindByID(context.Background(), productID)
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

	if response.Status != domain.OrderStatusPending {
		t.Errorf("status = %v, esperado %v", response.Status, domain.OrderStatusPending)
	}
	if response.Total != 30 {
		t.Errorf("total = %v, esperado 30", response.Total)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}

	product, _ := f.productStock.FindByID(context.Background(), productID)
	if product.Stock != 7 {
		t.Errorf("estoque restante = %d, esperado 7", product.Stock)
	}
	if len(f.productStock.releaseCalls) != 0 {
		t.Errorf("não deveria ter havido compensação (Release) no fluxo de sucesso, houve %d chamada(s)", len(f.productStock.releaseCalls))
	}
}

func TestOrderService_Create_ReservaFalhaAbortaAntesDaTransacaoLocal(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	f.productStock.reserveErr = errors.New("stock-service indisponível")

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if err == nil {
		t.Fatal("esperava erro ao reservar estoque")
	}
	if f.pool.tx.committed {
		t.Error("transação local não deveria ter sido commitada quando a reserva falha")
	}
	if f.pool.tx.rolledBack {
		t.Error("transação local nunca deveria ter sido aberta quando a reserva falha antes do Begin")
	}
}

func TestOrderService_Create_ReservaFalhaDisparaLiberacaoDefensiva(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	f.productStock.reserveErr = errors.New("timeout esperando resposta da reserva de estoque")

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(1)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if err == nil {
		t.Fatal("esperava erro ao reservar estoque")
	}
	if len(f.productStock.releaseCalls) != 1 {
		t.Fatalf("esperava 1 chamada de liberação defensiva, obteve %d", len(f.productStock.releaseCalls))
	}
	if f.productStock.releaseCalls[0].productID != productID || f.productStock.releaseCalls[0].quantity != 1 {
		t.Errorf("chamada de liberação defensiva inesperada: %+v", f.productStock.releaseCalls[0])
	}
}

func TestOrderService_Create_FalhaAposReservaCompensaEstoqueJaReservado(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	commitErr := errors.New("falha de rede ao commitar")
	f.pool.tx.commitErr = commitErr

	request := dto.CreateOrderRequest{
		ClientID: clientID,
		Items:    []dto.CreateOrderItemRequest{{ProductID: productID, Quantity: intPtr(4)}},
	}

	_, err := f.service.Create(context.Background(), request)

	if !errors.Is(err, commitErr) {
		t.Errorf("esperava que o erro de commit fosse propagado, obteve: %v", err)
	}

	if len(f.productStock.releaseCalls) != 1 {
		t.Fatalf("esperava 1 chamada de compensação (Release), obteve %d", len(f.productStock.releaseCalls))
	}
	if f.productStock.releaseCalls[0].quantity != 4 {
		t.Errorf("quantidade compensada = %d, esperado 4", f.productStock.releaseCalls[0].quantity)
	}
	if len(f.productStock.reserveCalls) != 1 || f.productStock.releaseCalls[0].sagaID != f.productStock.reserveCalls[0].sagaID {
		t.Errorf("a liberação deveria usar o mesmo saga_id da reserva original (Etapa 10): reserve=%+v release=%+v",
			f.productStock.reserveCalls, f.productStock.releaseCalls)
	}

	product, _ := f.productStock.FindByID(context.Background(), productID)
	if product.Stock != 10 {
		t.Errorf("estoque após compensação = %d, esperado 10 (reserva desfeita)", product.Stock)
	}
}

func TestOrderService_Pay_HappyPath(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, _ := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 50)

	response, err := f.service.Pay(context.Background(), orderID)
	if err != nil {
		t.Fatalf("Pay retornou erro inesperado: %v", err)
	}
	if response.Status != domain.OrderStatusPaid {
		t.Errorf("status = %v, esperado %v", response.Status, domain.OrderStatusPaid)
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
		seedStatus domain.OrderStatus
		wantErr    error
	}{
		{"pedido já pago", domain.OrderStatusPaid, custom_errors.ErrOrderAlreadyPaid},
		{"pedido cancelado", domain.OrderStatusCanceled, custom_errors.ErrOrderCannotChangeStatus},
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
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 50)
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
	// Simula que 3 unidades já haviam sido reservadas na criação do pedido.
	if err := f.productStock.Reserve(context.Background(), uuid.New(), productID, 3); err != nil {
		t.Fatalf("setup do estoque falhou: %v", err)
	}
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 30)
	if _, err := f.itemRepo.Create(context.Background(), nil, domain.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 3, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}

	response, err := f.service.Cancel(context.Background(), orderID)
	if err != nil {
		t.Fatalf("Cancel retornou erro inesperado: %v", err)
	}
	if response.Status != domain.OrderStatusCanceled {
		t.Errorf("status = %v, esperado %v", response.Status, domain.OrderStatusCanceled)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}

	product, _ := f.productStock.FindByID(context.Background(), productID)
	if product.Stock != 10 {
		t.Errorf("estoque após estorno = %d, esperado 10 (estoque original restaurado)", product.Stock)
	}
}

func TestOrderService_Cancel_RegrasDeStatus(t *testing.T) {
	tests := []struct {
		name       string
		seedStatus domain.OrderStatus
		wantErr    error
	}{
		{"pedido já cancelado", domain.OrderStatusCanceled, custom_errors.ErrOrderAlreadyCanceled},
		{"pedido já pago", domain.OrderStatusPaid, custom_errors.ErrOrderCannotChangeStatus},
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
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 30)
	if _, err := f.itemRepo.Create(context.Background(), nil, domain.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 3, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}
	f.productStock.releaseErr = errors.New("falha ao estornar estoque")

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
	if order.Status != domain.OrderStatusPending {
		t.Errorf("status do pedido não deveria ter mudado após rollback, obteve %v", order.Status)
	}
}

func TestOrderService_UpdateStatus_RegrasInvalidas(t *testing.T) {
	tests := []struct {
		name         string
		seedStatus   domain.OrderStatus
		targetStatus domain.OrderStatus
		wantErr      error
	}{
		{"status alvo inválido", domain.OrderStatusPending, domain.OrderStatusPending, custom_errors.ErrInvalidOrderStatus},
		{"pedido já pago", domain.OrderStatusPaid, domain.OrderStatusCanceled, custom_errors.ErrOrderAlreadyPaid},
		{"pedido já cancelado", domain.OrderStatusCanceled, domain.OrderStatusPaid, custom_errors.ErrOrderAlreadyCanceled},
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
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 30)

	response, err := f.service.UpdateStatus(context.Background(), orderID, domain.OrderStatusPaid)

	if err != nil {
		t.Fatalf("UpdateStatus retornou erro inesperado: %v", err)
	}
	if response.Status != domain.OrderStatusPaid {
		t.Errorf("status = %v, esperado %v", response.Status, domain.OrderStatusPaid)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}
}

func TestOrderService_UpdateStatus_ParaCanceladoEstornaEstoque(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	if err := f.productStock.Reserve(context.Background(), uuid.New(), productID, 4); err != nil {
		t.Fatalf("setup do estoque falhou: %v", err)
	}
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 40)
	if _, err := f.itemRepo.Create(context.Background(), nil, domain.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 4, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}

	_, err := f.service.UpdateStatus(context.Background(), orderID, domain.OrderStatusCanceled)
	if err != nil {
		t.Fatalf("UpdateStatus retornou erro inesperado: %v", err)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}

	product, _ := f.productStock.FindByID(context.Background(), productID)
	if product.Stock != 10 {
		t.Errorf("estoque após estorno = %d, esperado 10", product.Stock)
	}
}

func TestOrderService_FindByID_HappyPath(t *testing.T) {
	f := newOrderServiceFixture()
	clientID, productID := setupClientAndProduct(f, 10)
	orderID := seedOrder(f, clientID, domain.OrderStatusPending, 30)
	if _, err := f.itemRepo.Create(context.Background(), nil, domain.OrderItem{OrderID: orderID, ProductID: productID, Quantity: 3, Price: 10}); err != nil {
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
	firstID := seedOrder(f, clientID, domain.OrderStatusPending, 10)
	secondID := seedOrder(f, clientID, domain.OrderStatusPaid, 20)
	if _, err := f.itemRepo.Create(context.Background(), nil, domain.OrderItem{OrderID: firstID, ProductID: productID, Quantity: 1, Price: 10}); err != nil {
		t.Fatalf("setup do item falhou: %v", err)
	}
	if _, err := f.itemRepo.Create(context.Background(), nil, domain.OrderItem{OrderID: secondID, ProductID: productID, Quantity: 2, Price: 10}); err != nil {
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
