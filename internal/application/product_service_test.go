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

type fakeProductRepository struct {
	byName         map[string]domain.Product
	created        []domain.Product
	findByNameErr  error
	updateErr      error
	updateStockErr error
}

func newFakeProductRepository() *fakeProductRepository {
	return &fakeProductRepository{byName: make(map[string]domain.Product)}
}

func (f *fakeProductRepository) Create(ctx context.Context, product domain.Product) (domain.Product, error) {
	product.ID = uuid.New()
	f.created = append(f.created, product)
	f.byName[product.Name] = product
	return product, nil
}

func (f *fakeProductRepository) FindAll(ctx context.Context) ([]domain.Product, error) {
	products := make([]domain.Product, 0, len(f.byName))
	for _, p := range f.byName {
		products = append(products, p)
	}
	return products, nil
}

func (f *fakeProductRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	for _, p := range f.byName {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Product{}, custom_errors.ErrProductNotFound
}

func (f *fakeProductRepository) FindByIDForUpdate(ctx context.Context, tx Tx, id uuid.UUID) (domain.Product, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeProductRepository) FindByName(ctx context.Context, name string) (domain.Product, error) {
	if f.findByNameErr != nil {
		return domain.Product{}, f.findByNameErr
	}
	product, ok := f.byName[name]
	if !ok {
		return domain.Product{}, custom_errors.ErrProductNotFound
	}
	return product, nil
}

func (f *fakeProductRepository) Update(ctx context.Context, tx Tx, id uuid.UUID, product domain.Product) (domain.Product, error) {
	if f.updateErr != nil {
		return domain.Product{}, f.updateErr
	}

	existing, err := f.FindByID(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}

	product.ID = existing.ID
	if existing.Name != product.Name {
		delete(f.byName, existing.Name)
	}
	f.byName[product.Name] = product

	return product, nil
}

func (f *fakeProductRepository) UpdateStock(ctx context.Context, tx Tx, productID uuid.UUID, delta int) error {
	if f.updateStockErr != nil {
		return f.updateStockErr
	}

	for name, product := range f.byName {
		if product.ID != productID {
			continue
		}

		newStock := product.Stock + delta
		if newStock < 0 {
			return custom_errors.ErrInsufficientStock
		}

		product.Stock = newStock
		f.byName[name] = product
		return nil
	}

	return custom_errors.ErrProductNotFound
}

func (f *fakeProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	for name, p := range f.byName {
		if p.ID == id {
			delete(f.byName, name)
			return nil
		}
	}
	return custom_errors.ErrProductNotFound
}

func newProductServiceForTest(repo ProductRepository) *ProductService {
	return NewProductService(nil, repo)
}

type productServiceFixture struct {
	pool    *fakeConnPool
	repo    *fakeProductRepository
	service *ProductService
}

func newProductServiceFixture() *productServiceFixture {
	f := &productServiceFixture{
		pool: newFakeConnPool(),
		repo: newFakeProductRepository(),
	}
	f.service = NewProductService(f.pool, f.repo)
	return f
}

func validUpdateProductRequest() dto.UpdateProductRequest {
	price := 6000.0
	stock := 5
	return dto.UpdateProductRequest{
		Name:  "Notebook Pro",
		Price: &price,
		Stock: &stock,
	}
}

func validCreateProductRequest() dto.CreateProductRequest {
	price := 5000.50
	stock := 10
	return dto.CreateProductRequest{
		Name:  "Notebook",
		Price: &price,
		Stock: &stock,
	}
}

func TestProductService_Create_HappyPath(t *testing.T) {
	repo := newFakeProductRepository()
	service := newProductServiceForTest(repo)

	response, err := service.Create(context.Background(), validCreateProductRequest())
	if err != nil {
		t.Fatalf("Create retornou erro inesperado: %v", err)
	}

	if response.Name != "Notebook" || response.Stock != 10 {
		t.Errorf("resposta inesperada: %+v", response)
	}
}

func TestProductService_Create_Validacoes(t *testing.T) {
	price := 10.0
	stock := 1

	tests := []struct {
		name    string
		mutate  func(r *dto.CreateProductRequest)
		wantErr error
	}{
		{"nome vazio", func(r *dto.CreateProductRequest) { r.Name = "" }, custom_errors.ErrProductNameRequired},
		{"preço nulo", func(r *dto.CreateProductRequest) { r.Price = nil }, custom_errors.ErrProductPriceRequired},
		{"preço zero", func(r *dto.CreateProductRequest) { z := 0.0; r.Price = &z }, custom_errors.ErrProductPriceInvalid},
		{"preço negativo", func(r *dto.CreateProductRequest) { n := -5.0; r.Price = &n }, custom_errors.ErrProductPriceInvalid},
		{"estoque nulo", func(r *dto.CreateProductRequest) { r.Stock = nil }, custom_errors.ErrProductStockRequired},
		{"estoque negativo", func(r *dto.CreateProductRequest) { n := -1; r.Stock = &n }, custom_errors.ErrProductStockInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeProductRepository()
			service := newProductServiceForTest(repo)

			request := dto.CreateProductRequest{Name: "Produto", Price: &price, Stock: &stock}
			tt.mutate(&request)

			_, err := service.Create(context.Background(), request)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestProductService_Create_NomeJaExiste(t *testing.T) {
	repo := newFakeProductRepository()
	service := newProductServiceForTest(repo)

	request := validCreateProductRequest()
	if _, err := service.Create(context.Background(), request); err != nil {
		t.Fatalf("criação inicial falhou: %v", err)
	}

	_, err := service.Create(context.Background(), request)

	if !errors.Is(err, custom_errors.ErrProductNameExists) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNameExists)
	}
}

func TestProductService_FindByID_ProdutoInexistente(t *testing.T) {
	repo := newFakeProductRepository()
	service := newProductServiceForTest(repo)

	_, err := service.FindByID(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}

func TestProductService_Delete_ProdutoInexistente(t *testing.T) {
	repo := newFakeProductRepository()
	service := newProductServiceForTest(repo)

	err := service.Delete(context.Background(), uuid.New())

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}

func TestProductService_FindAll_HappyPath(t *testing.T) {
	repo := newFakeProductRepository()
	service := newProductServiceForTest(repo)

	if _, err := service.Create(context.Background(), validCreateProductRequest()); err != nil {
		t.Fatalf("criação inicial falhou: %v", err)
	}

	response, err := service.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll retornou erro inesperado: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("esperava 1 produto, obteve %d", len(response))
	}
}

func TestProductService_Update_Validacoes(t *testing.T) {
	price := 10.0
	stock := 1

	tests := []struct {
		name    string
		mutate  func(r *dto.UpdateProductRequest)
		wantErr error
	}{
		{"nome vazio", func(r *dto.UpdateProductRequest) { r.Name = "" }, custom_errors.ErrProductNameRequired},
		{"preço nulo", func(r *dto.UpdateProductRequest) { r.Price = nil }, custom_errors.ErrProductPriceRequired},
		{"preço zero", func(r *dto.UpdateProductRequest) { z := 0.0; r.Price = &z }, custom_errors.ErrProductPriceInvalid},
		{"preço negativo", func(r *dto.UpdateProductRequest) { n := -5.0; r.Price = &n }, custom_errors.ErrProductPriceInvalid},
		{"estoque nulo", func(r *dto.UpdateProductRequest) { r.Stock = nil }, custom_errors.ErrProductStockRequired},
		{"estoque negativo", func(r *dto.UpdateProductRequest) { n := -1; r.Stock = &n }, custom_errors.ErrProductStockInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			f := newProductServiceFixture()

			request := dto.UpdateProductRequest{Name: "Produto", Price: &price, Stock: &stock}
			tt.mutate(&request)

			_, err := f.service.Update(context.Background(), uuid.New(), request)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestProductService_Update_HappyPath(t *testing.T) {
	f := newProductServiceFixture()

	created, err := f.repo.Create(context.Background(), domain.Product{Name: "Notebook", Price: 5000, Stock: 10})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	response, err := f.service.Update(context.Background(), created.ID, validUpdateProductRequest())
	if err != nil {
		t.Fatalf("Update retornou erro inesperado: %v", err)
	}
	if response.Name != "Notebook Pro" || response.Stock != 5 {
		t.Errorf("resposta inesperada: %+v", response)
	}
	if !f.pool.tx.committed {
		t.Error("esperava que a transação fosse commitada no fluxo de sucesso")
	}
}

func TestProductService_Update_ProdutoInexistente(t *testing.T) {
	f := newProductServiceFixture()

	_, err := f.service.Update(context.Background(), uuid.New(), validUpdateProductRequest())

	if !errors.Is(err, custom_errors.ErrProductNotFound) {
		t.Errorf("erro = %v, esperado %v", err, custom_errors.ErrProductNotFound)
	}
}

func TestProductService_Update_RollbackQuandoUpdateFalha(t *testing.T) {
	f := newProductServiceFixture()

	created, err := f.repo.Create(context.Background(), domain.Product{Name: "Notebook", Price: 5000, Stock: 10})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	f.repo.updateErr = errors.New("falha ao atualizar produto")

	_, err = f.service.Update(context.Background(), created.ID, validUpdateProductRequest())

	if err == nil {
		t.Fatal("esperava erro ao atualizar produto")
	}
	if f.pool.tx.committed {
		t.Error("transação não deveria ter sido commitada quando Update falha")
	}
	if !f.pool.tx.rolledBack {
		t.Error("esperava rollback da transação quando Update falha")
	}
}
