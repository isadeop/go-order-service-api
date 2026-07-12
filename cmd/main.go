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

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	routes.ClientRoutes(
		r,
		clientController,
	)

	log.Printf(
		"API rodando em http://localhost:%s",
		cfg.Port,
	)

	log.Println("POST   /client       ->  create client | criar cliente")
	log.Println("GET    /client       -> list clients | listar clientes")
	log.Println("GET    /cliente/{id}  -> search by id | buscar por id")

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
