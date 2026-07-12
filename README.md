# go-order-service-api
# Go Order Service API

API REST simples desenvolvida em Go como atividade avaliativa de fechamento do primeiro módulo em GO.
Tema: gerenciamento de clientes, produtos e pedidos.

O projeto foi desenvolvido com foco em aplicar conceitos de arquitetura em camadas (Controller → Service → Repository), validações de regras de negócio, transações com PostgreSQL e organização do código. Não foram implementadas medidas de segurança tendo em vista o objetivo geral de familiarização.

## Tecnologias

- Go
- PostgreSQL
- Chi Router
- pgx/v5
- golang-migrate
- UUID (google/uuid)

---

## Funcionalidades

### Clientes

- Criar cliente
- Listar clientes
- Buscar cliente por ID

### Produtos

- Criar produto
- Listar produtos
- Buscar produto por ID
- Atualizar produto
- Remover produto

### Pedidos

- Criar pedido
- Listar pedidos com paginação (`limit` e `offset`)
- Buscar pedido por ID
- Alterar status do pedido
- Pagar pedido
- Cancelar pedido
- Atualização automática de estoque ao criar ou cancelar pedidos

---

# Pré-requisitos

- Go 1.24+
- PostgreSQL
- golang-migrate

Instalação do migrate:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

# Banco de dados

Crie um banco PostgreSQL.

Exemplo:

```text
Database: orderserviceapi

User: adm

Password: adm
```

Ajuste a string de conexão conforme necessário.

---

# Executando as migrations

Aplicar todas as migrations:

```bash
migrate -database "postgres://adm:adm@localhost:5432/orderserviceapi?sslmode=disable" -path migrations up
```

Remover todas as migrations:

```bash
migrate -database "postgres://adm:adm@localhost:5432/orderserviceapi?sslmode=disable" -path migrations down -all
```

---

# Executando o projeto

Instale as dependências:

```bash
go mod tidy
```

Execute:

```bash
go run cmd/main.go
```

A API iniciará em:

```
http://localhost:8080
```

---

# Rotas

## Clientes

| Método | Endpoint | Descrição |
|---------|----------|-----------|
| POST | `/clientes` | Criar cliente |
| GET | `/clientes` | Listar clientes |
| GET | `/clientes/{id}` | Buscar cliente por ID |

---

## Produtos

| Método | Endpoint | Descrição |
|---------|----------|-----------|
| POST | `/produtos` | Criar produto |
| GET | `/produtos` | Listar produtos |
| GET | `/produtos/{id}` | Buscar produto |
| PUT | `/produtos/{id}` | Atualizar produto |
| DELETE | `/produtos/{id}` | Remover produto |

---

## Pedidos

| Método | Endpoint | Descrição |
|---------|----------|-----------|
| POST | `/pedidos` | Criar pedido |
| GET | `/pedidos?limit=10&offset=0` | Listar pedidos |
| GET | `/pedidos/{id}` | Buscar pedido |
| PATCH | `/pedidos/{id}/status` | Alterar status do pedido |
| POST | `/pedidos/{id}/pagar` | Marcar pedido como pago |
| POST | `/pedidos/{id}/cancelar` | Cancelar pedido |

---

# Testando a API

A API pode ser testada utilizando ferramentas como:

- Insomnia
- Postman
- Bruno
- cURL

---

# Estrutura do projeto

```
cmd/
internal/
    config/
    controllers/
    custom_errors/
    database/
    dto/
    model/
    repository/
    routes/
    services/
    migrations/
    security/
```

---

# Regras de negócio implementadas

- Não permite criar pedidos sem cliente.
- Não permite criar pedidos sem itens.
- Não permite quantidade menor ou igual a zero.
- Verifica existência do cliente.
- Verifica existência do produto.
- Verifica estoque disponível.
- Atualiza o estoque ao criar pedidos.
- Devolve o estoque ao cancelar pedidos.
- Não permite cancelar pedidos pagos.
- Não permite pagar pedidos já cancelados.
- Não permite alterar pedidos já finalizados.

---

# Possíveis melhorias

Este projeto foi desenvolvido com fins de estudo e demonstração. Algumas melhorias que podem ser implementadas incluem:

- Autenticação (JWT)
- Autorização por perfis (RBAC)
- Documentação da API (OpenAPI/Swagger)
- Testes unitários e de integração
- Redução de repetição de código entre Services e Controllers
- Centralização do tratamento de erros (middleware)
- Logging estruturado
- Cache para consultas frequentes
- Paginação padronizada
- Filtros de busca
- CI/CD
- Docker e Docker Compose
- Outros
---

# Json

## Criar cliente (`POST /clientes`)

```json
{
  "name": "João Silva",
  "email": "joao.silva@email.com",
  "phone": "11999999999",
  "password": "123456"
}
```

---

## Criar produto (`POST /produtos`)

```json
{
  "name": "Notebook",
  "price": 5000.50,
  "stock": 10
}
```
`

---

## Criar pedido (`POST /pedidos`)

Substitua os UUIDs pelos retornados nos endpoints de cliente e produto.

```json
{
  "client_id": "UUID_DO_CLIENTE",
  "items": [
    {
      "product_id": "UUID_DO_NOTEBOOK",
      "quantity": 2
    },
    {
      "product_id": "UUID_DO_MOUSE",
      "quantity": 1
    }
  ]
}
```

---
