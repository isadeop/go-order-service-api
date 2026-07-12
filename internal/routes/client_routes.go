package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/isadeop/go-order-service-api/internal/controllers"
)

func ClientRoutes(r chi.Router, controller *controllers.ClientController) {
	r.Post("/clients", controller.CreateClient)
	r.Get("/clients", controller.FindAllClients)
	r.Get("/clients/{id}", controller.FindClientByID)
}
