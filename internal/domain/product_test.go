package domain

import (
	"errors"
	"testing"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
)

func TestProduct_Reserve(t *testing.T) {
	tests := []struct {
		name      string
		stock     int
		quantity  int
		wantErr   error
		wantStock int
	}{
		{"reserva parcial do estoque", 10, 3, nil, 7},
		{"reserva o estoque inteiro", 10, 10, nil, 0},
		{"quantidade maior que o estoque disponível", 5, 6, custom_errors.ErrInsufficientStock, 5},
		{"quantidade zero é inválida", 10, 0, custom_errors.ErrOrderItemQuantityInvalid, 10},
		{"quantidade negativa é inválida", 10, -1, custom_errors.ErrOrderItemQuantityInvalid, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := Product{Stock: tt.stock}

			err := product.Reserve(tt.quantity)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, esperado %v", err, tt.wantErr)
			}

			if product.Stock != tt.wantStock {
				t.Errorf("estoque = %d, esperado %d", product.Stock, tt.wantStock)
			}
		})
	}
}

func TestProduct_Reserve_ChamadasSucessivas(t *testing.T) {
	product := Product{Stock: 5}

	if err := product.Reserve(3); err != nil {
		t.Fatalf("primeira reserva não deveria falhar: %v", err)
	}

	err := product.Reserve(3)
	if !errors.Is(err, custom_errors.ErrInsufficientStock) {
		t.Fatalf("erro = %v, esperado %v", err, custom_errors.ErrInsufficientStock)
	}

	if product.Stock != 2 {
		t.Errorf("estoque = %d, esperado 2 (só a primeira reserva deve ter sido aplicada)", product.Stock)
	}
}

func TestProduct_Release(t *testing.T) {
	product := Product{Stock: 2}

	product.Release(3)

	if product.Stock != 5 {
		t.Errorf("estoque = %d, esperado 5", product.Stock)
	}
}
