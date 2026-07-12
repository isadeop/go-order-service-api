package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/isadeop/go-order-service-api/internal/controllers"
)

func OrderRoutes(r chi.Router, controller *controllers.OrderController) {
	r.Post("/orders", controller.CreateOrder)
	r.Get("/orders", controller.FindOrders)
	r.Get("/orders/{id}", controller.FindOrderByID)
	r.Patch("/orders/{id}/status", controller.UpdateOrderStatus)
}
