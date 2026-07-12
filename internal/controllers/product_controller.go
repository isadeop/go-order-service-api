package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type ProductService interface {
	Create(ctx context.Context, request dto.CreateProductRequest) (dto.ProductResponse, error)
	FindAll(ctx context.Context) ([]dto.ProductResponse, error)
	FindByID(ctx context.Context, id uuid.UUID) (dto.ProductResponse, error)
	Update(ctx context.Context, id uuid.UUID, request dto.UpdateProductRequest) (dto.ProductResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductController struct {
	service ProductService
}

func NewProductController(service ProductService) *ProductController {
	return &ProductController{
		service: service,
	}
}

func writeProductError(w http.ResponseWriter, err error) {

	switch {
	case errors.Is(err, custom_errors.ErrInvalidProductID):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrProductNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, custom_errors.ErrProductNameExists):
		http.Error(w, err.Error(), http.StatusConflict)

	case errors.Is(err, custom_errors.ErrProductNameRequired),
		errors.Is(err, custom_errors.ErrProductPriceRequired),
		errors.Is(err, custom_errors.ErrProductStockRequired),
		errors.Is(err, custom_errors.ErrProductPriceInvalid),
		errors.Is(err, custom_errors.ErrProductStockInvalid):

		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		log.Printf("internal error: %v", err)
		http.Error(w, "internal error / erro interno do servidor", http.StatusInternalServerError)
	}
}

func (c *ProductController) CreateProduct(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request dto.CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}

	response, err := c.service.Create(
		r.Context(),
		request,
	)

	if err != nil {
		log.Println("erro ao criar produto:", err)
		writeProductError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) FindAllProducts(
	w http.ResponseWriter,
	r *http.Request,
) {

	response, err := c.service.FindAll(r.Context())

	if err != nil {
		writeProductError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) FindProductByID(
	w http.ResponseWriter,
	r *http.Request,
) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		writeProductError(w, custom_errors.ErrInvalidProductID)
		return
	}

	response, err := c.service.FindByID(
		r.Context(),
		id,
	)

	if err != nil {
		writeProductError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) UpdateProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProductError(w, custom_errors.ErrInvalidProductID)
		return
	}

	var request dto.UpdateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}

	response, err := c.service.Update(
		r.Context(),
		id,
		request,
	)

	if err != nil {
		writeProductError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) DeleteProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProductError(w, custom_errors.ErrInvalidProductID)
		return
	}

	err = c.service.Delete(
		r.Context(),
		id,
	)

	if err != nil {
		writeProductError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
