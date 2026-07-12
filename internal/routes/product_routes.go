package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/isadeop/go-order-service-api/internal/controllers"
)

func ProductRoutes(r chi.Router, controller *controllers.ProductController) {
	r.Post("/produtos", controller.CreateProduct)
	r.Get("/produtos", controller.FindAllProducts)
	r.Get("/produtos/{id}", controller.FindProductByID)
	r.Put("/produtos/{id}", controller.UpdateProduct)
	r.Delete("/produtos/{id}", controller.DeleteProduct)
}
