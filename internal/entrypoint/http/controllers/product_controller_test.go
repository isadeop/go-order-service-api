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
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type fakeProductService struct {
	createResponse  dto.ProductResponse
	createErr       error
	findAllResp     []dto.ProductResponse
	findAllErr      error
	findByIDErr     error
	updateResponse  dto.ProductResponse
	updateErr       error
	reserveResponse dto.ProductResponse
	reserveErr      error
	releaseResponse dto.ProductResponse
	releaseErr      error
	deleteErr       error
}

func (f *fakeProductService) Create(ctx context.Context, request dto.CreateProductRequest) (dto.ProductResponse, error) {
	return f.createResponse, f.createErr
}

func (f *fakeProductService) FindAll(ctx context.Context) ([]dto.ProductResponse, error) {
	return f.findAllResp, f.findAllErr
}

func (f *fakeProductService) FindByID(ctx context.Context, id uuid.UUID) (dto.ProductResponse, error) {
	return dto.ProductResponse{}, f.findByIDErr
}

func (f *fakeProductService) Update(ctx context.Context, id uuid.UUID, request dto.UpdateProductRequest) (dto.ProductResponse, error) {
	return f.updateResponse, f.updateErr
}

func (f *fakeProductService) Reserve(ctx context.Context, id uuid.UUID, quantity int) (dto.ProductResponse, error) {
	return f.reserveResponse, f.reserveErr
}

func (f *fakeProductService) Release(ctx context.Context, id uuid.UUID, quantity int) (dto.ProductResponse, error) {
	return f.releaseResponse, f.releaseErr
}

func (f *fakeProductService) Delete(ctx context.Context, id uuid.UUID) error {
	return f.deleteErr
}

func newProductRouter(service ProductService) http.Handler {
	controller := NewProductController(service)
	r := chi.NewRouter()
	r.Post("/produtos", controller.CreateProduct)
	r.Get("/produtos", controller.FindAllProducts)
	r.Get("/produtos/{id}", controller.FindProductByID)
	r.Put("/produtos/{id}", controller.UpdateProduct)
	r.Post("/produtos/{id}/reservar", controller.Reserve)
	r.Post("/produtos/{id}/liberar", controller.Release)
	r.Delete("/produtos/{id}", controller.DeleteProduct)
	return r
}

func TestProductController_CreateProduct_HappyPath(t *testing.T) {
	service := &fakeProductService{createResponse: dto.ProductResponse{Name: "Notebook", Stock: 10}}
	router := newProductRouter(service)

	body := `{"name":"Notebook","price":5000.5,"stock":10}`
	req := httptest.NewRequest(http.MethodPost, "/produtos", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusCreated)
	}
}

// TestProductController_CreateProduct_ErrosDoService cobre como o controller
// traduz cada erro de negócio do service em status HTTP.
func TestProductController_CreateProduct_ErrosDoService(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{"nome já existe", `{"name":"Notebook","price":5000.5,"stock":10}`, custom_errors.ErrProductNameExists, http.StatusConflict},
		{"estoque inválido", `{"name":"Notebook","price":5000.5,"stock":-1}`, custom_errors.ErrProductStockInvalid, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newProductRouter(&fakeProductService{createErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/produtos", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestProductController_FindProductByID_IDInvalido(t *testing.T) {
	router := newProductRouter(&fakeProductService{})

	req := httptest.NewRequest(http.MethodGet, "/produtos/nao-e-um-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProductController_FindAllProducts_HappyPath(t *testing.T) {
	service := &fakeProductService{findAllResp: []dto.ProductResponse{{ID: uuid.New(), Name: "Notebook", Stock: 10}}}
	router := newProductRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/produtos", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestProductController_FindProductByID_NaoEncontrado(t *testing.T) {
	service := &fakeProductService{findByIDErr: custom_errors.ErrProductNotFound}
	router := newProductRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/produtos/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestProductController_UpdateProduct_HappyPath(t *testing.T) {
	service := &fakeProductService{updateResponse: dto.ProductResponse{Name: "Notebook Pro", Stock: 5}}
	router := newProductRouter(service)

	body := `{"name":"Notebook Pro","price":6000,"stock":5}`
	req := httptest.NewRequest(http.MethodPut, "/produtos/"+uuid.New().String(), bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestProductController_Reserve_HappyPath(t *testing.T) {
	service := &fakeProductService{reserveResponse: dto.ProductResponse{Name: "Notebook", Stock: 7}}
	router := newProductRouter(service)

	body := `{"quantity":3}`
	req := httptest.NewRequest(http.MethodPost, "/produtos/"+uuid.New().String()+"/reservar", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestProductController_Reserve_EstoqueInsuficiente(t *testing.T) {
	service := &fakeProductService{reserveErr: custom_errors.ErrInsufficientStock}
	router := newProductRouter(service)

	body := `{"quantity":100}`
	req := httptest.NewRequest(http.MethodPost, "/produtos/"+uuid.New().String()+"/reservar", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusConflict)
	}
}

func TestProductController_Release_HappyPath(t *testing.T) {
	service := &fakeProductService{releaseResponse: dto.ProductResponse{Name: "Notebook", Stock: 10}}
	router := newProductRouter(service)

	body := `{"quantity":3}`
	req := httptest.NewRequest(http.MethodPost, "/produtos/"+uuid.New().String()+"/liberar", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}
}

func TestProductController_Release_ProdutoNaoEncontrado(t *testing.T) {
	service := &fakeProductService{releaseErr: custom_errors.ErrProductNotFound}
	router := newProductRouter(service)

	body := `{"quantity":1}`
	req := httptest.NewRequest(http.MethodPost, "/produtos/"+uuid.New().String()+"/liberar", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestProductController_DeleteProduct_NaoEncontrado(t *testing.T) {
	service := &fakeProductService{deleteErr: custom_errors.ErrProductNotFound}
	router := newProductRouter(service)

	req := httptest.NewRequest(http.MethodDelete, "/produtos/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestProductController_DeleteProduct_HappyPath(t *testing.T) {
	service := &fakeProductService{}
	router := newProductRouter(service)

	req := httptest.NewRequest(http.MethodDelete, "/produtos/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNoContent)
	}
}
