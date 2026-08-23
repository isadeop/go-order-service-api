package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/domain"
)

func TestClientRepository_Create_HappyPath(t *testing.T) {
	pool := newTestPool(t)

	client := createTestClient(t, pool)

	if client.ID == uuid.Nil {
		t.Error("esperava que o ID fosse preenchido pelo banco")
	}
	if client.CreatedAt.IsZero() || client.UpdatedAt.IsZero() {
		t.Error("esperava created_at/updated_at preenchidos pelo banco")
	}
}

func TestClientRepository_Create_EmailDuplicado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	existing := createTestClient(t, pool)

	_, err := repo.Create(context.Background(), domain.Client{
		Name:         "Outro Nome",
		Email:        existing.Email,
		Phone:        "11888888888",
		PasswordHash: "outro-hash",
	})

	if !errors.Is(err, custom_errors.ErrClientEmailAlreadyExists) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientEmailAlreadyExists)
	}
}

func TestClientRepository_FindByID_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientNotFound)
	}
}

func TestClientRepository_FindByID_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	created := createTestClient(t, pool)

	found, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID retornou erro inesperado: %v", err)
	}
	if found.Email != created.Email {
		t.Errorf("email = %q, esperado %q", found.Email, created.Email)
	}
}

func TestClientRepository_FindByEmail_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	_, err := repo.FindByEmail(context.Background(), "nao-cadastrado-"+uuid.NewString()+"@teste.local")

	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientNotFound)
	}
}

func TestClientRepository_FindByEmail_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	created := createTestClient(t, pool)

	found, err := repo.FindByEmail(context.Background(), created.Email)
	if err != nil {
		t.Fatalf("FindByEmail retornou erro inesperado: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, esperado %v", found.ID, created.ID)
	}
}

func TestClientRepository_FindAll_ContemClienteCriado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	created := createTestClient(t, pool)

	clients, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}

	found := false
	for _, c := range clients {
		if c.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("esperava encontrar o cliente recém-criado na listagem")
	}
}

func TestClientRepository_Update_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	created := createTestClient(t, pool)
	newEmail := "atualizado-" + uuid.NewString() + "@integracao.local"

	updated, err := repo.Update(context.Background(), created.ID, domain.Client{
		Name:  "Nome Atualizado",
		Email: newEmail,
		Phone: "11977777777",
	})
	if err != nil {
		t.Fatalf("Update retornou erro inesperado: %v", err)
	}
	if updated.Name != "Nome Atualizado" || updated.Email != newEmail {
		t.Errorf("cliente atualizado inesperado: %+v", updated)
	}
}

func TestClientRepository_Update_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	_, err := repo.Update(context.Background(), uuid.New(), domain.Client{
		Name:  "Nome",
		Email: "inexistente-" + uuid.NewString() + "@teste.local",
		Phone: "11999999999",
	})

	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientNotFound)
	}
}

func TestClientRepository_Delete_HappyPath(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	client, err := repo.Create(context.Background(), domain.Client{
		Name:         "Cliente Para Deletar",
		Email:        "deletar-" + uuid.NewString() + "@integracao.local",
		Phone:        "11999999999",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := repo.Delete(context.Background(), client.ID); err != nil {
		t.Fatalf("Delete retornou erro inesperado: %v", err)
	}

	_, err = repo.FindByID(context.Background(), client.ID)
	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		t.Errorf("esperava que o cliente não existisse mais após Delete, FindByID retornou: %v", err)
	}
}

func TestClientRepository_Delete_NaoEncontrado(t *testing.T) {
	pool := newTestPool(t)
	repo := NewClientRepository(pool)

	err := repo.Delete(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrClientNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrClientNotFound)
	}
}
