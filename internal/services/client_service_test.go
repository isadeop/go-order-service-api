package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
	"github.com/isadeop/go-order-service-api/internal/model"
)

type fakeClientRepository struct {
	byEmail        map[string]model.Client
	created        []model.Client
	createErr      error
	findByEmailErr error
}

func newFakeClientRepository() *fakeClientRepository {
	return &fakeClientRepository{byEmail: make(map[string]model.Client)}
}

func (f *fakeClientRepository) Create(ctx context.Context, client model.Client) (model.Client, error) {
	if f.createErr != nil {
		return model.Client{}, f.createErr
	}
	client.ID = uuid.New()
	f.created = append(f.created, client)
	f.byEmail[client.Email] = client
	return client, nil
}

func (f *fakeClientRepository) FindAll(ctx context.Context) ([]model.Client, error) {
	clients := make([]model.Client, 0, len(f.byEmail))
	for _, c := range f.byEmail {
		clients = append(clients, c)
	}
	return clients, nil
}

func (f *fakeClientRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Client, error) {
	for _, c := range f.byEmail {
		if c.ID == id {
			return c, nil
		}
	}
	return model.Client{}, custom_errors.ErrClientNotFound
}

func (f *fakeClientRepository) FindByEmail(ctx context.Context, email string) (model.Client, error) {
	if f.findByEmailErr != nil {
		return model.Client{}, f.findByEmailErr
	}
	client, ok := f.byEmail[email]
	if !ok {
		return model.Client{}, custom_errors.ErrClientNotFound
	}
	return client, nil
}

func validCreateClientRequest() dto.CreateClientRequest {
	return dto.CreateClientRequest{
		Name:     "João Silva",
		Email:    "joao@example.com",
		Phone:    "11999999999",
		Password: "12345678",
	}
}

func newClientServiceForTest(repo ClientRepository) *ClientService {
	return &ClientService{repository: repo}
}

func TestClientService_Create_HappyPath(t *testing.T) {
	repo := newFakeClientRepository()
	service := newClientServiceForTest(repo)

	response, err := service.Create(context.Background(), validCreateClientRequest())
	if err != nil {
		t.Fatalf("Create retornou erro inesperado: %v", err)
	}

	if response.Email != "joao@example.com" {
		t.Errorf("email da resposta = %q, esperado %q", response.Email, "joao@example.com")
	}

	if len(repo.created) != 1 {
		t.Fatalf("esperava 1 cliente persistido, obteve %d", len(repo.created))
	}

	if repo.created[0].PasswordHash == "" || repo.created[0].PasswordHash == "12345678" {
		t.Error("a senha deveria ter sido hasheada antes de persistir")
	}
}

func TestClientService_Create_CamposObrigatorios(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(r *dto.CreateClientRequest)
		wantErr error
	}{
		{"nome vazio", func(r *dto.CreateClientRequest) { r.Name = "" }, custom_errors.ErrClientNameRequired},
		{"email vazio", func(r *dto.CreateClientRequest) { r.Email = "" }, custom_errors.ErrClientEmailRequired},
		{"telefone vazio", func(r *dto.CreateClientRequest) { r.Phone = "" }, custom_errors.ErrClientPhoneRequired},
		{"senha vazia", func(r *dto.CreateClientRequest) { r.Password = "" }, custom_errors.ErrClientPasswordRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeClientRepository()
			service := newClientServiceForTest(repo)

			request := validCreateClientRequest()
			tt.mutate(&request)

			_, err := service.Create(context.Background(), request)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientService_Create_EmailJaExiste(t *testing.T) {
	repo := newFakeClientRepository()
	service := newClientServiceForTest(repo)

	first := validCreateClientRequest()
	if _, err := service.Create(context.Background(), first); err != nil {
		t.Fatalf("criação inicial falhou: %v", err)
	}

	_, err := service.Create(context.Background(), first)

	if !errors.Is(err, custom_errors.ErrClientEmailAlreadyExists) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientEmailAlreadyExists)
	}
}

func TestClientService_Create_ErroInesperadoNaBuscaPorEmail(t *testing.T) {
	repo := newFakeClientRepository()
	repo.findByEmailErr = errors.New("falha de conexão com o banco")
	service := newClientServiceForTest(repo)

	_, err := service.Create(context.Background(), validCreateClientRequest())

	if err == nil || errors.Is(err, custom_errors.ErrClientEmailAlreadyExists) {
		t.Errorf("esperava que o erro de infraestrutura fosse propagado, obteve: %v", err)
	}
}

func TestClientService_FindByID_ClienteInexistente(t *testing.T) {
	repo := newFakeClientRepository()
	service := newClientServiceForTest(repo)

	_, err := service.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientNotFound)
	}
}

func TestClientService_Create_ErroInesperadoAoPersistir(t *testing.T) {
	repo := newFakeClientRepository()
	repo.createErr = errors.New("falha de conexão ao inserir")
	service := newClientServiceForTest(repo)

	_, err := service.Create(context.Background(), validCreateClientRequest())

	if err == nil || !errors.Is(err, repo.createErr) {
		t.Errorf("esperava que o erro de infraestrutura fosse propagado, obteve: %v", err)
	}
}

func TestClientService_FindAll_HappyPath(t *testing.T) {
	repo := newFakeClientRepository()
	service := newClientServiceForTest(repo)

	if _, err := service.Create(context.Background(), validCreateClientRequest()); err != nil {
		t.Fatalf("criação inicial falhou: %v", err)
	}

	response, err := service.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("esperava 1 cliente, obteve %d", len(response))
	}
}

func TestClientService_FindAll_PropagaErroDoRepository(t *testing.T) {
	infraErr := errors.New("falha de conexão")
	service := newClientServiceForTest(&erroringClientRepository{err: infraErr})

	_, err := service.FindAll(context.Background())

	if !errors.Is(err, infraErr) {
		t.Errorf("esperava que o erro de infraestrutura fosse propagado, obteve: %v", err)
	}
}
