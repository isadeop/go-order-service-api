package custom_errors

import "errors"

var ErrOrderNotFound = errors.New("order not found / pedido não encontrado")
var ErrOrderClientRequired = errors.New("order client is required / cliente do pedido é obrigatório")
var ErrOrderClientNotFound = errors.New("order client not found / cliente do pedido não encontrado")
var ErrOrderItemsRequired = errors.New("order must have at least one item / pedido deve possuir pelo menos um item")
var ErrOrderProductNotFound = errors.New("order product not found / produto do pedido não encontrado")
var ErrInsufficientStock = errors.New("insufficient product stock / estoque insuficiente do produto")
var ErrInvalidOrderID = errors.New("order id must be a valid uuid / id do pedido deve ser um uuid válido")
var ErrInvalidOrderStatus = errors.New("invalid order status / status do pedido inválido")
var ErrOrderAlreadyPaid = errors.New("order is already paid / pedido já está pago")
var ErrOrderAlreadyCanceled = errors.New("order is already canceled / pedido já está cancelado")
var ErrOrderCannotChangeStatus = errors.New("paid or canceled orders cannot change status / pedidos pagos ou cancelados não podem alterar o status")
var ErrOrderPaymentFailed = errors.New("order payment failed / pagamento do pedido falhou")
var ErrOrderCancellationFailed = errors.New("order cancellation failed / cancelamento do pedido falhou")
var ErrInvalidPagination = errors.New("limit and offset must be valid positive integers / limit e offset devem ser inteiros válidos e positivos")
