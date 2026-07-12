package custom_errors

import "errors"

var ErrOrderItemNotFound = errors.New("order item not found / item do pedido não encontrado")
var ErrOrderItemQuantityRequired = errors.New("order item quantity is required / quantidade do item do pedido é obrigatória")
var ErrOrderItemQuantityInvalid = errors.New("order item quantity must be greater than zero / quantidade do item do pedido deve ser maior que zero")
var ErrOrderItemProductRequired = errors.New("order item product is required / produto do item do pedido é obrigatório")
var ErrOrderItemOrderRequired = errors.New("order item order is required / pedido do item é obrigatório")
var ErrInvalidOrderItemID = errors.New("order item id must be a valid uuid / id do item do pedido deve ser um uuid válido")
