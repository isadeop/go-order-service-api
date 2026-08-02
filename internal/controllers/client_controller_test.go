package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type fakeClientService struct {
	createResponse dto.ClientResponse
	createErr      error
	findByIDErr    error
	findByIDResp   dto.ClientResponse
	findAllResp    []dto.ClientResponse
	findAllErr     error
}

func (f *fakeClientService) Create(ctx context.Context, request dto.CreateClientRequest) (dto.ClientResponse, error) {
	return f.createResponse, f.createErr
}

func (f *fakeClientService) FindAll(ctx context.Context) ([]dto.ClientResponse, error) {
	return f.findAllResp, f.findAllErr
}

func (f *fakeClientService) FindByID(ctx context.Context, id uuid.UUID) (dto.ClientResponse, error) {
	return f.findByIDResp, f.findByIDErr
}

func newClientRouter(service ClientService) http.Handler {
	controller := NewClientController(service)
	r := chi.NewRouter()
	r.Post("/clientes", controller.CreateClient)
	r.Get("/clientes", controller.FindAllClients)
	r.Get("/clientes/{id}", controller.FindClientByID)
	return r
}

func TestClientController_CreateClient_HappyPath(t *testing.T) {
	service := &fakeClientService{createResponse: dto.ClientResponse{ID: uuid.New(), Name: "João"}}
	router := newClientRouter(service)

	body := `{"name":"João","email":"joao@example.com","phone":"11999999999","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/clientes", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusCreated)
	}

	var response dto.ClientResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("resposta não é um JSON válido: %v", err)
	}
	if response.Name != "João" {
		t.Errorf("name = %q, esperado %q", response.Name, "João")
	}
}

func TestClientController_CreateClient_JSONInvalido(t *testing.T) {
	router := newClientRouter(&fakeClientService{})

	req := httptest.NewRequest(http.MethodPost, "/clientes", bytes.NewBufferString("{ isto não é json"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

// TestClientController_CreateClient_ErrosDoService cobre como o controller
// traduz alguns erros de negócio do service em status HTTP.
func TestClientController_CreateClient_ErrosDoService(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{"email já existe", `{"name":"João","email":"joao@example.com","phone":"11999999999","password":"12345678"}`, custom_errors.ErrClientEmailAlreadyExists, http.StatusConflict},
		{"campo obrigatório faltando", `{}`, custom_errors.ErrClientNameRequired, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newClientRouter(&fakeClientService{createErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/clientes", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestClientController_FindClientByID_NaoEncontrado(t *testing.T) {
	service := &fakeClientService{findByIDErr: custom_errors.ErrClientNotFound}
	router := newClientRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/clientes/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}

func TestClientController_FindClientByID_IDInvalido(t *testing.T) {
	router := newClientRouter(&fakeClientService{})

	req := httptest.NewRequest(http.MethodGet, "/clientes/nao-e-um-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
}

func TestClientController_FindAllClients_HappyPath(t *testing.T) {
	service := &fakeClientService{findAllResp: []dto.ClientResponse{{ID: uuid.New(), Name: "João"}}}
	router := newClientRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/clientes", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}

	var response []dto.ClientResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("resposta não é um JSON válido: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("esperava 1 cliente, obteve %d", len(response))
	}
}
