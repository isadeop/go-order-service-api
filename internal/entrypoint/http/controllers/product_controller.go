package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
	"github.com/isadeop/go-order-service-api/internal/dto"
)

type ProductService interface {
	Create(ctx context.Context, request dto.CreateProductRequest) (dto.ProductResponse, error)
	FindAll(ctx context.Context) ([]dto.ProductResponse, error)
	FindByID(ctx context.Context, id uuid.UUID) (dto.ProductResponse, error)
	Update(ctx context.Context, id uuid.UUID, request dto.UpdateProductRequest) (dto.ProductResponse, error)
	Reserve(ctx context.Context, id uuid.UUID, quantity int) (dto.ProductResponse, error)
	Release(ctx context.Context, id uuid.UUID, quantity int) (dto.ProductResponse, error)
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

func writeProductError(w http.ResponseWriter, operation string, err error) {

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
		errors.Is(err, custom_errors.ErrProductStockInvalid),
		errors.Is(err, custom_errors.ErrOrderItemQuantityInvalid):

		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, custom_errors.ErrInsufficientStock):
		http.Error(w, err.Error(), http.StatusConflict)

	default:
		slog.Error("product.internal_error",
			"operation", operation,
			"result", "error",
			"err", err.Error(),
		)
		http.Error(w, "internal error / erro interno do servidor", http.StatusInternalServerError)
	}
}

func (c *ProductController) CreateProduct(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request dto.CreateProductRequest

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.Create(
		r.Context(),
		request,
	)

	if err != nil {
		slog.Warn("product.create_failed",
			"operation", "CreateProduct",
			"result", "error",
			"err", err.Error(),
		)
		writeProductError(w, "CreateProduct", err)
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
		writeProductError(w, "FindAllProducts", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) FindProductByID(
	w http.ResponseWriter,
	r *http.Request,
) {

	id, err := parseIDParam(r)

	if err != nil {
		writeProductError(w, "FindProductByID", custom_errors.ErrInvalidProductID)
		return
	}

	response, err := c.service.FindByID(
		r.Context(),
		id,
	)

	if err != nil {
		writeProductError(w, "FindProductByID", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) UpdateProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parseIDParam(r)
	if err != nil {
		writeProductError(w, "UpdateProduct", custom_errors.ErrInvalidProductID)
		return
	}

	var request dto.UpdateProductRequest

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.Update(
		r.Context(),
		id,
		request,
	)

	if err != nil {
		writeProductError(w, "UpdateProduct", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// reserveOrReleaseRequest é o payload de /produtos/{id}/reservar e
// /produtos/{id}/liberar — a única informação necessária é a quantidade.
type reserveOrReleaseRequest struct {
	Quantity int `json:"quantity"`
}

func (c *ProductController) Reserve(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parseIDParam(r)
	if err != nil {
		writeProductError(w, "Reserve", custom_errors.ErrInvalidProductID)
		return
	}

	var request reserveOrReleaseRequest

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.Reserve(
		r.Context(),
		id,
		request.Quantity,
	)

	if err != nil {
		writeProductError(w, "Reserve", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) Release(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parseIDParam(r)
	if err != nil {
		writeProductError(w, "Release", custom_errors.ErrInvalidProductID)
		return
	}

	var request reserveOrReleaseRequest

	if !decodeJSONBody(w, r, &request) {
		return
	}

	response, err := c.service.Release(
		r.Context(),
		id,
		request.Quantity,
	)

	if err != nil {
		writeProductError(w, "Release", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *ProductController) DeleteProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parseIDParam(r)
	if err != nil {
		writeProductError(w, "DeleteProduct", custom_errors.ErrInvalidProductID)
		return
	}

	err = c.service.Delete(
		r.Context(),
		id,
	)

	if err != nil {
		writeProductError(w, "DeleteProduct", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
