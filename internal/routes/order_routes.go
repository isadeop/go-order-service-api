package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/isadeop/go-order-service-api/internal/controllers"
)

func OrderRoutes(r chi.Router, controller *controllers.OrderController) {
	r.Post("/pedidos", controller.CreateOrder)
	r.Get("/pedidos", controller.FindOrders)
	r.Get("/pedidos/{id}", controller.FindOrderByID)
	r.Post("/pedidos/{id}/pagar", controller.Pay)
	r.Post("/pedidos/{id}/cancelar", controller.Cancel)
	r.Patch("/pedidos/{id}/status", controller.UpdateOrderStatus)
}
