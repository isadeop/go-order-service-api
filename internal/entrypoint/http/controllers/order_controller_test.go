package controllers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type fakeOrderService struct {
	createResponse   dto.OrderResponse
	createErr        error
	findByIDResp     dto.OrderResponse
	findByIDErr      error
	findAllCalled    bool
	findAllLimit     int
	findAllOffset    int
	findAllErr       error
	updateStatusResp dto.OrderResponse
	updateStatusErr  error
	payResp          dto.OrderResponse
	payErr           error
	cancelErr        error
}

func (f *fakeOrderService) Create(ctx context.Context, request dto.CreateOrderRequest) (dto.OrderResponse, error) {
	return f.createResponse, f.createErr
}

func (f *fakeOrderService) FindByID(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {
	return f.findByIDResp, f.findByIDErr
}

func (f *fakeOrderService) FindAll(ctx context.Context, limit int, offset int) ([]dto.OrderResponse, error) {
	f.findAllCalled = true
	f.findAllLimit = limit
	f.findAllOffset = offset
	return nil, f.findAllErr
}

func (f *fakeOrderService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (dto.OrderResponse, error) {
	return f.updateStatusResp, f.updateStatusErr
}

func (f *fakeOrderService) Pay(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {
	return f.payResp, f.payErr
}

func (f *fakeOrderService) Cancel(ctx context.Context, id uuid.UUID) (dto.OrderResponse, error) {
	return dto.OrderResponse{}, f.cancelErr
}

func newOrderRouter(service OrderService) (http.Handler, *fakeOrderService) {
	controller := NewOrderController(service)
	r := chi.NewRouter()
	r.Post("/pedidos", controller.CreateOrder)
	r.Get("/pedidos", controller.FindOrders)
	r.Get("/pedidos/{id}", controller.FindOrderByID)
	r.Post("/pedidos/{id}/pagar", controller.Pay)
	r.Post("/pedidos/{id}/cancelar", controller.Cancel)
	r.Patch("/pedidos/{id}/status", controller.UpdateOrderStatus)
	return r, service.(*fakeOrderService)
}

func TestOrderController_CreateOrder_HappyPath(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{createResponse: dto.OrderResponse{ID: uuid.New()}})

	body := `{"client_id":"` + uuid.New().String() + `","items":[{"product_id":"` + uuid.New().String() + `","quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/pedidos", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusCreated)
	}
}

// TestOrderController_CreateOrder_ErrosDoService cobre como o controller
// traduz cada erro de negócio do service em status HTTP.
func TestOrderController_CreateOrder_ErrosDoService(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{"cliente inexistente", custom_errors.ErrOrderClientNotFound, http.StatusNotFound},
		{"produto inexistente deve ser 404, não 400", custom_errors.ErrOrderProductNotFound, http.StatusNotFound},
		{"estoque insuficiente", custom_errors.ErrInsufficientStock, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := newOrderRouter(&fakeOrderService{createErr: tt.serviceErr})

			body := `{"client_id":"` + uuid.New().String() + `","items":[{"product_id":"` + uuid.New().String() + `","quantity":1}]}`
			req := httptest.NewRequest(http.MethodPost, "/pedidos", bytes.NewBufferString(body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestOrderController_Pay_ErrosDoService(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{"pedido já cancelado", custom_errors.ErrOrderCannotChangeStatus, http.StatusConflict},
		{"pedido inexistente", custom_errors.ErrOrderNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := newOrderRouter(&fakeOrderService{payErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/pedidos/"+uuid.New().String()+"/pagar", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestOrderController_Cancel_ErrosDoService(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{"pedido já cancelado", custom_errors.ErrOrderAlreadyCanceled, http.StatusConflict},
		{"pedido inexistente", custom_errors.ErrOrderNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := newOrderRouter(&fakeOrderService{cancelErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/pedidos/"+uuid.New().String()+"/cancelar", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestOrderController_FindOrders_PaginacaoValida(t *testing.T) {
	router, service := newOrderRouter(&fakeOrderService{})

	req := httptest.NewRequest(http.MethodGet, "/pedidos?limit=5&offset=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
	if !service.findAllCalled {
		t.Fatal("esperava que o service.FindAll fosse chamado")
	}
	if service.findAllLimit != 5 || service.findAllOffset != 10 {
		t.Errorf("limit/offset = %d/%d, esperado 5/10", service.findAllLimit, service.findAllOffset)
	}
}

func TestOrderController_FindOrders_PaginacaoInvalida(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"limit não numérico", "limit=abc"},
		{"limit zero", "limit=0"},
		{"limit negativo", "limit=-5"},
		{"offset não numérico", "offset=xyz"},
		{"offset negativo", "offset=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, service := newOrderRouter(&fakeOrderService{})

			req := httptest.NewRequest(http.MethodGet, "/pedidos?"+tt.query, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
			}
			if service.findAllCalled {
				t.Error("paginação inválida não deveria chegar a chamar o service")
			}
		})
	}
}

func TestOrderController_FindOrderByID_IDInvalido(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{})

	req := httptest.NewRequest(http.MethodGet, "/pedidos/nao-e-um-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOrderController_FindOrderByID_HappyPath(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{findByIDResp: dto.OrderResponse{ID: uuid.New(), Status: domain.OrderStatusPending}})

	req := httptest.NewRequest(http.MethodGet, "/pedidos/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestOrderController_FindOrderByID_PedidoInexistente(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{findByIDErr: custom_errors.ErrOrderNotFound})

	req := httptest.NewRequest(http.MethodGet, "/pedidos/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestOrderController_Pay_HappyPath(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{payResp: dto.OrderResponse{ID: uuid.New(), Status: domain.OrderStatusPaid}})

	req := httptest.NewRequest(http.MethodPost, "/pedidos/"+uuid.New().String()+"/pagar", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestOrderController_Pay_PedidoInexistente(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{payErr: custom_errors.ErrOrderNotFound})

	req := httptest.NewRequest(http.MethodPost, "/pedidos/"+uuid.New().String()+"/pagar", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestOrderController_UpdateOrderStatus_HappyPath(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{updateStatusResp: dto.OrderResponse{ID: uuid.New(), Status: domain.OrderStatusCanceled}})

	body := `{"status":"CANCELED"}`
	req := httptest.NewRequest(http.MethodPatch, "/pedidos/"+uuid.New().String()+"/status", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestOrderController_UpdateOrderStatus_StatusInvalido(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{updateStatusErr: custom_errors.ErrInvalidOrderStatus})

	body := `{"status":"QUALQUER_COISA"}`
	req := httptest.NewRequest(http.MethodPatch, "/pedidos/"+uuid.New().String()+"/status", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOrderController_UpdateOrderStatus_IDInvalido(t *testing.T) {
	router, _ := newOrderRouter(&fakeOrderService{})

	body := `{"status":"PAID"}`
	req := httptest.NewRequest(http.MethodPatch, "/pedidos/nao-e-um-uuid/status", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}
