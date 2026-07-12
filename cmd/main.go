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

	orderRepository :=
		repository.NewOrderRepository(pool)

	orderItemRepository :=
		repository.NewOrderItemRepository(pool)

	productStockRepository :=
		repository.NewProductRepository(pool)

	orderService :=
		services.NewOrderService(
			pool,
			orderRepository,
			orderItemRepository,
			productStockRepository,
			clientRepository,
		)

	orderController :=
		controllers.NewOrderController(orderService)

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

	routes.OrderRoutes(
		r,
		orderController,
	)

	log.Printf(
		"API rodando em http://localhost:%s",
		cfg.Port,
	)

	log.Println("POST   /clientes       ->  create client | criar cliente")
	log.Println("GET    /clientes       -> list clients | listar clientes")
	log.Println("GET    /clientes/{id}  -> search by id | buscar por id")

	log.Println("POST   /produtos       ->  create product | criar produto")
	log.Println("GET    /produtos       -> list products | listar produtos")
	log.Println("GET    /produtos/{id}  -> search product by id | buscar produto por id")
	log.Println("PUT    /produtos/{id}  -> update product by id | atualizar produto por id")
	log.Println("DELETE    /produtos/{id}  -> delete product by id | deletar produto por id")

	log.Println("POST   /pedidos       ->  create order | criar pedido")
	log.Println("GET    /pedidos?limit=10&offset=0       -> list order | listar pedidos")
	log.Println("GET    /pedidos/{id}  -> search order by id | buscar pedido por id")
	log.Println("PATCH   /pedidos/{id}/status  -> update order status by id | atualizar status do pedido por id")
	log.Println("POST   /pedidos/{id}/pagar       ->  update order status to paid | atualizar status do pedido para paid (pago)")
	log.Println("POST   /pedidos/{id}/cancelar       ->  update order status to canceled | atualizar status do pedido para canceled (cancelado)")

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
