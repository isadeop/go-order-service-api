package main

import (
	"context"
	"log"
	"net/http"

	"github.com/isadeop/go-order-service-api/internal/config"
	"github.com/isadeop/go-order-service-api/internal/controllers"
	"github.com/isadeop/go-order-service-api/internal/database"
	"github.com/isadeop/go-order-service-api/internal/repository"
	"github.com/isadeop/go-order-service-api/internal/routes"
	"github.com/isadeop/go-order-service-api/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	ctx := context.Background()

	cfg := config.Load()

	pool, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)

	if err != nil {
		log.Fatalf(
			"error connecting with database | erro ao conectar no banco: %v",
			err,
		)
	}

	defer pool.Close()

	clientRepository :=
		repository.NewClientRepository(pool)

	clientService :=
		services.NewClientService(clientRepository)

	clientController :=
		controllers.NewClientController(clientService)

	productRepository :=
		repository.NewProductRepository(pool)

	productService :=
		services.NewProductService(productRepository)

	productController :=
		controllers.NewProductController(productService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	routes.ClientRoutes(
		r,
		clientController,
	)

	routes.ProductRoutes(
		r,
		productController,
	)

	log.Printf(
		"API rodando em http://localhost:%s",
		cfg.Port,
	)

	log.Println("POST   /client       ->  create client | criar cliente")
	log.Println("GET    /client       -> list clients | listar clientes")
	log.Println("GET    /cliente/{id}  -> search by id | buscar por id")

	log.Println("POST   /products       ->  create product | criar produto")
	log.Println("GET    /products       -> list products | listar produtos")
	log.Println("GET    /products/{id}  -> search by id | buscar por id")
	log.Println("PUT    /products/{id}  -> update product by id | atualizar produto por id")
	log.Println("DELETE    /products/{id}  -> delete product by id | deletar produto por id")

	if err := http.ListenAndServe(
		":"+cfg.Port,
		r,
	); err != nil {

		log.Fatalf(
			"error initializing server | erro ao iniciar servidor: %v",
			err,
		)
	}
}
