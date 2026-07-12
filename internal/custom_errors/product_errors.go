package custom_errors

import "errors"

var ErrProductNotFound = errors.New("product not found / produto não encontrado")
var ErrProductNameRequired = errors.New("product name is required / nome do produto é obrigatório")
var ErrProductNameExists = errors.New("product name already registered / nome de produto já registrado")
var ErrProductPriceRequired = errors.New("product price is required / preço do produto é obrigatório")
var ErrProductPriceInvalid = errors.New("product price must be greater than zero / preço do produto deve ser maior que zero")
var ErrProductStockInvalid = errors.New("product stock cannot be negative / estoque do produto não pode ser negativo")
var ErrProductStockRequired = errors.New("product stock is required / estoque do produto é obrigatório")
var ErrInvalidProductID = errors.New("product id must be a valid uuid / id do produto deve ser um uuid válido")
