# go-order-service-api
# Go Order Service API

API REST simples desenvolvida em Go como atividade avaliativa de fechamento do primeiro e segundo módulos em GO.
Tema: gerenciamento de clientes, produtos e pedidos.

O projeto foi desenvolvido com foco em aplicar conceitos de arquitetura em camadas (Controller → Service → Repository), validações de regras de negócio, transações com PostgreSQL e organização do código. Não foram implementadas medidas de segurança tendo em vista o objetivo geral de familiarização.

Módulo 1: avaliação do projeto;
Módulo 2: avaliação de implementação de testes e correções ou melhorias pontuadas pelo professor;

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
migrate -database "postgres://adm:adm@localhost:5432/orderserviceapi?sslmode=disable" -path internal/migrations up
```

Remover todas as migrations:

```bash
migrate -database "postgres://adm:adm@localhost:5432/orderserviceapi?sslmode=disable" -path internal/migrations down -all
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

# Testes

O projeto possui:

- Testes unitários dos services, cobrindo as regras de negócio (criação, pagamento, cancelamento, validações), sem depender de banco.
- Testes de integração dos repositories, contra um PostgreSQL real.
- Testes dos controllers, cobrindo a tradução de cada erro de negócio em status HTTP.
- Testes de concorrência, validando que operações críticas de pedido (criação e cancelamento) mantêm o estoque e o status consistentes sob concorrência (sem lost update, sem estorno duplicado).

Comandos
Rodar toda a suíte:

```bash
go test ./...
```

Ver cobertura:

```bash
go test ./... -cover
```

Cobertura atual por camada:

| Pacote | Cobertura |
|---|---|
| `services` | ~87% |
| `controllers` | ~82% |
| `repository` | ~82% |
| `security` | ~80% |

`model`, `dto`, `routes`, `config`, `database` e `cmd` não têm testes próprios pois não apresentam regra de negócio.

Os testes de `repository` e os de concorrência precisam de um PostgreSQL acessível (o mesmo configurado em `.env`/variáveis de ambiente). Se o banco não estiver disponível, esses testes são pulados automaticamente, sem quebrar `go test ./...`.

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
