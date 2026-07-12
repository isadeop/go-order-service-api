package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/isadeop/go-order-service-api/internal/controllers"
)

func ProductRoutes(r chi.Router, controller *controllers.ProductController) {
	r.Post("/products", controller.CreateProduct)
	r.Get("/products", controller.FindAllProducts)
	r.Get("/products/{id}", controller.FindProductByID)
	r.Put("/products/{id}", controller.UpdateProduct)
	r.Delete("/products/{id}", controller.DeleteProduct)
}
