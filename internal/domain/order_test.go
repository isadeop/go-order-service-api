package domain

import (
	"errors"
	"testing"

	"github.com/isadeop/go-order-service-api/internal/custom_errors"
)

func TestOrder_Pay(t *testing.T) {
	tests := []struct {
		name       string
		seedStatus OrderStatus
		wantErr    error
	}{
		{"pedido pendente pode ser pago", OrderStatusPending, nil},
		{"pedido já pago não pode ser pago de novo", OrderStatusPaid, custom_errors.ErrOrderAlreadyPaid},
		{"pedido cancelado não pode ser pago", OrderStatusCanceled, custom_errors.ErrOrderCannotChangeStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Status: tt.seedStatus}

			err := order.Pay()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, esperado %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if order.Status != OrderStatusPaid {
					t.Errorf("status = %v, esperado %v", order.Status, OrderStatusPaid)
				}
			} else if order.Status != tt.seedStatus {
				t.Errorf("status não deveria mudar quando Pay falha, obteve %v", order.Status)
			}
		})
	}
}

func TestOrder_Cancel(t *testing.T) {
	tests := []struct {
		name       string
		seedStatus OrderStatus
		wantErr    error
	}{
		{"pedido pendente pode ser cancelado", OrderStatusPending, nil},
		{"pedido já cancelado não pode ser cancelado de novo", OrderStatusCanceled, custom_errors.ErrOrderAlreadyCanceled},
		{"pedido pago não pode ser cancelado", OrderStatusPaid, custom_errors.ErrOrderCannotChangeStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Status: tt.seedStatus}

			err := order.Cancel()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, esperado %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if order.Status != OrderStatusCanceled {
					t.Errorf("status = %v, esperado %v", order.Status, OrderStatusCanceled)
				}
			} else if order.Status != tt.seedStatus {
				t.Errorf("status não deveria mudar quando Cancel falha, obteve %v", order.Status)
			}
		})
	}
}

func TestOrder_ChangeStatus(t *testing.T) {
	tests := []struct {
		name         string
		seedStatus   OrderStatus
		targetStatus OrderStatus
		wantErr      error
	}{
		{"pendente para pago", OrderStatusPending, OrderStatusPaid, nil},
		{"pendente para cancelado", OrderStatusPending, OrderStatusCanceled, nil},
		{"já pago não muda, mesmo que o alvo seja cancelado", OrderStatusPaid, OrderStatusCanceled, custom_errors.ErrOrderAlreadyPaid},
		{"já cancelado não muda, mesmo que o alvo seja pago", OrderStatusCanceled, OrderStatusPaid, custom_errors.ErrOrderAlreadyCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Status: tt.seedStatus}

			err := order.ChangeStatus(tt.targetStatus)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, esperado %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if order.Status != tt.targetStatus {
					t.Errorf("status = %v, esperado %v", order.Status, tt.targetStatus)
				}
			} else if order.Status != tt.seedStatus {
				t.Errorf("status não deveria mudar quando ChangeStatus falha, obteve %v", order.Status)
			}
		})
	}
}
