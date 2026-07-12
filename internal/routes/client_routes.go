package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/isadeop/go-order-service-api/internal/controllers"
)

func ClientRoutes(r chi.Router, controller *controllers.ClientController) {
	r.Post("/clientes", controller.CreateClient)
	r.Get("/clientes", controller.FindAllClients)
	r.Get("/clientes/{id}", controller.FindClientByID)
}
