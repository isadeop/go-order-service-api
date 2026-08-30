# go-order-service-api
# Go Order Service API

API desenvolvida em Go como atividade avaliativa dos módulos de Go. Tema: gerenciamento de clientes, produtos e pedidos.

O projeto nasceu como um monolito (camadas Controller → Service → Repository, um único banco) e foi evoluído, num módulo seguinte, para uma arquitetura de microsserviços com DDD/Clean Architecture.
Cada serviço é dono do seu próprio banco, apresenta comunicação síncrona (HTTP) e assíncrona (mensageria, padrão Saga) entre eles, e logging estruturado. Não foram implementadas medidas de segurança de produção (autenticação/autorização, rate limiting, etc.) tendo em vista o objetivo geral de familiarização com os conceitos — ver [Possíveis melhorias](#possíveis-melhorias).

---

## Arquitetura

Dois serviços independentes, cada um dono do seu próprio banco PostgreSQL:

- **order-service** (`cmd/order-service`) — clientes e pedidos (`orders_db`).
- **stock-service** (`cmd/stock-service`) — produtos e estoque (`stock_db`).

O order-service não acessa o banco de produtos diretamente. Ele fala com o stock-service de duas formas:

- **HTTP síncrono** — consulta de produto (`FindByID`) e liberação de estoque (`Release`, usada em cancelamento/estorno).
- **Mensageria assíncrona (Saga)** — a reserva de estoque na criação de um pedido é feita publicando um comando no Redpanda (Kafka-compatible) e aguardando a resposta correlacionada, num padrão de requisição/resposta sobre fila. Isso preserva o contrato HTTP síncrono de `POST /pedidos` (o cliente da API não percebe que por trás existe troca de mensagens) enquanto o desacoplamento entre os dois serviços passa a valer para a operação mais sensível a concorrência (reserva de estoque).

```
Cliente HTTP
     │
     ▼
┌─────────────────┐        stock-commands (Redpanda)        ┌─────────────────┐
│  order-service   │ ───────────────────────────────────▶   │  stock-service  │
│  (orders_db)     │                                          │  (stock_db)     │
│                  │ ◀───────────────────────────────────    │                 │
└─────────────────┘        stock-events (Redpanda)          └─────────────────┘
     │        ▲                                                      ▲
     │        │              HTTP (FindByID, Release)                │
     │        └──────────────────────────────────────────────────────┘
     ▼
 orders_db                                                       stock_db
```

### Saga de criação de pedido

1. `POST /pedidos` valida a requisição e busca os produtos (HTTP) no stock-service.
2. Para cada produto do pedido, o order-service publica `stock.reserve.requested` em `stock-commands`, correlacionado por `saga_id` (um UUID gerado por tentativa de criação de pedido).
3. O stock-service consome o comando, reserva o estoque  e publica `stock.reserved` ou `stock.reservation.failed` em `stock-events`.
4. O order-service, que ficou aguardando essa resposta específica com timeout, segue para criar o pedido localmente — ou, se algum item falhar, compensa (libera) os itens já reservados.

**Idempotência**: cada reserva fica registrada (`stock_reservations`, chave `(saga_id, product_id)`) no `stock_db`. Isso torna tanto a reserva quanto a liberação seguras de repetir.

**Falha e reconciliação**: se a resposta de uma reserva demorar mais que o timeout do order-service, ele libera defensivamente o item (idempotente, então seguro mesmo sem saber se a reserva realmente foi aplicada). Ainda assim, existe uma janela de corrida: se a reserva for aplicada pelo stock-service *depois* dessa liberação defensiva, ela fica "órfã" (sem pedido correspondente, e sem ninguém que ainda esteja esperando por ela). Para fechar esse gap, o stock-service roda uma  reconciliação periódica que libera reservas `RESERVED` mais antigas —  [Limitações conhecidas](#limitações-conhecidas).

---

## Tecnologias

- Go 1.26
- PostgreSQL 18+ (bancos separados por serviço; migrations usam `uuidv7()` como default de UUID)
- Redpanda (broker compatível com o protocolo Kafka) + [franz-go](https://github.com/twmb/franz-go)
- Chi Router
- pgx/v5
- golang-migrate
- Docker / Docker Compose
- `log/slog` (logging estruturado em JSON)
- UUID (google/uuid)

---

## Funcionalidades

### order-service

**Clientes**
- Criar cliente
- Listar clientes
- Buscar cliente por ID

**Pedidos**
- Criar pedido (dispara a Saga de reserva de estoque)
- Listar pedidos com paginação (`limit` e `offset`)
- Buscar pedido por ID
- Alterar status do pedido
- Pagar pedido
- Cancelar pedido (devolve o estoque reservado)

### stock-service

**Produtos**
- Criar produto
- Listar produtos
- Buscar produto por ID
- Atualizar produto
- Remover produto
- Reservar/liberar estoque (via HTTP, para chamadas diretas/teste — ou via Saga, para o fluxo real de criação de pedido)
- Reconciliação periódica de reservas órfãs

---

## Pré-requisitos

**Com Docker** — Docker e Docker Compose. Ele sobe os dois bancos, o Redpanda, aplica as migrations e inicia os dois serviços.

**Sem Docker** (rodar `go run`/`go test` direto no host):

- Go 1.26+
- Dois PostgreSQL 18+ (um para `orders_db`, outro para `stock_db`) — ou um único servidor com dois databases
- Um broker Redpanda/Kafka acessível
- golang-migrate

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## Executando com Docker

```bash
docker compose up --build
```

Isso sobe, na ordem correta de dependências:

- `postgres-orders` / `postgres-stock` — um Postgres 18 para cada serviço.
- `migrate-orders` / `migrate-stock` — aplicam as migrations de cada banco e saem.
- `redpanda` — o broker de mensageria (modo dev, um nó só).
- `redpanda-console` — UI web opcional para inspecionar tópicos/mensagens.
- `order-service` / `stock-service` — os dois serviços da API.

| Serviço | URL |
|---|---|
| order-service (clientes, pedidos) | `http://localhost:8080` |
| stock-service (produtos) | `http://localhost:8081` |
| Redpanda Console | `http://localhost:8090` |
| postgres-orders (host) | `localhost:5434` |
| postgres-stock (host) | `localhost:5433` |
| Redpanda (Kafka API, host) | `localhost:19092` |

Para derrubar tudo (e apagar os dados dos bancos):

```bash
docker compose down -v
```

Para rodar só a infraestrutura (bancos + Redpanda) e a API/testes direto no host:

```bash
docker compose up -d postgres-orders postgres-stock migrate-orders migrate-stock redpanda
```

---

## Executando localmente (sem Docker)

Aplique as migrations de cada banco separadamente:

```bash
migrate -database "postgres://adm:adm@localhost:5434/orders_db?sslmode=disable" -path internal/migrations/orders up
migrate -database "postgres://adm:adm@localhost:5433/stock_db?sslmode=disable" -path internal/migrations/stock up
```

Instale as dependências e rode cada serviço num terminal:

```bash
go mod tidy
go run cmd/order-service/main.go
go run cmd/stock-service/main.go
```

Configuração via variáveis de ambiente (ou um arquivo `.env` na raiz — veja a tabela abaixo).

### Variáveis de ambiente

| Variável | Usado por | Padrão | Descrição |
|---|---|---|---|
| `PORT` | ambos | `8080` | Porta HTTP do serviço |
| `POSTGRES_HOST` / `POSTGRES_PORT` / `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` / `POSTGRES_SSLMODE` | ambos | `localhost` / `5432` / `adm` / `adm` / *(nome do banco do serviço)* / `disable` | Conexão com o Postgres do próprio serviço |
| `STOCK_SERVICE_URL` | order-service | `http://localhost:8081` | Base URL HTTP do stock-service |
| `REDPANDA_BROKERS` | ambos | `localhost:19092` | Lista de brokers (separados por vírgula) |
| `STOCK_RESERVATION_STALE_AFTER` | stock-service | `10m` | A partir de quanto tempo `RESERVED` uma reserva é considerada órfã pelo job de reconciliação |
| `STOCK_RESERVATION_SWEEP_INTERVAL` | stock-service | `1m` | Frequência do job de reconciliação |

---

## Rotas

### order-service — Clientes

| Método | Endpoint | Descrição |
|---------|----------|-----------|
| POST | `/clientes` | Criar cliente |
| GET | `/clientes` | Listar clientes |
| GET | `/clientes/{id}` | Buscar cliente por ID |

### order-service — Pedidos

| Método | Endpoint | Descrição |
|---------|----------|-----------|
| POST | `/pedidos` | Criar pedido (dispara a Saga de reserva de estoque) |
| GET | `/pedidos?limit=10&offset=0` | Listar pedidos |
| GET | `/pedidos/{id}` | Buscar pedido |
| PATCH | `/pedidos/{id}/status` | Alterar status do pedido |
| POST | `/pedidos/{id}/pagar` | Marcar pedido como pago |
| POST | `/pedidos/{id}/cancelar` | Cancelar pedido |

### stock-service — Produtos

| Método | Endpoint | Descrição |
|---------|----------|-----------|
| POST | `/produtos` | Criar produto |
| GET | `/produtos` | Listar produtos |
| GET | `/produtos/{id}` | Buscar produto |
| PUT | `/produtos/{id}` | Atualizar produto |
| POST | `/produtos/{id}/reservar` | Reservar estoque (chamada direta/teste — o fluxo real de pedido usa a Saga via Redpanda) |
| POST | `/produtos/{id}/liberar` | Liberar estoque (usada pelo order-service em cancelamento/compensação) |
| DELETE | `/produtos/{id}` | Remover produto |

---

## Testando a API

- Insomnia
- Postman
- Bruno
- cURL

---

## Testes

O projeto possui:

- Testes unitários de domínio (`domain`), cobrindo as invariantes de estado do pedido e do estoque (`Order.Pay/Cancel`, `Product.Reserve/Release`), sem depender de banco nem de HTTP.
- Testes unitários de aplicação (`application`), cobrindo os casos de uso (criação, pagamento, cancelamento, reserva/liberação idempotente, reconciliação, validações) com fakes de repository, sem depender de banco.
- Testes de integração de `infra/repository`, contra PostgreSQL real (os dois bancos).
- Testes dos controllers (`entrypoint/http/controllers`), cobrindo a tradução de cada erro de negócio em status HTTP.
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
| `application` | ~83% |
| `domain` | 100% |
| `entrypoint/http/controllers` | ~82% |
| `infra/repository` | ~81% |
| `security` | ~80% |

`dto`, `entrypoint/http/routes`, `entrypoint/messaging/consumers`, `infra/config`, `infra/database`, `infra/messaging`, `infra/sagaclient`, `infra/stockclient`, `observability`, `txport` e `cmd` não têm testes próprios diretos.

Os testes de `infra/repository`, os de concorrência e os que envolvem mensageria precisam de PostgreSQL (os dois bancos) e Redpanda acessíveis. Se a infraestrutura não estiver disponível, esses testes são pulados automaticamente (`t.Skip`), sem quebrar `go test ./...`.

---

## Estrutura do projeto

```
cmd/
    order-service/             # entrypoint do order-service
    stock-service/             # entrypoint do stock-service
internal/
    domain/                    # entidades e invariantes (Order, Product, StockReservation...)
    application/                # casos de uso, orquestra domínio + portas (OrderService, ProductService...)
    entrypoint/
        http/
            controllers/        # handlers HTTP
            routes/              # registro das rotas
        messaging/
            consumers/           # tradução de mensagens Redpanda em chamadas de caso de uso
    infra/
        repository/              # implementação concreta dos repositories (pgx/SQL)
        database/                # criação do pool de conexões (pgxpool)
        config/                  # leitura de variáveis
        messaging/               # envelope de mensagem, producer, criação de tópicos
        sagaclient/              # lado order-service da troca de reserva (Saga request/reply)
        stockclient/             # cliente HTTP do order-service para o stock-service
    custom_errors/               # erros de domínio
    dto/
    txport/                      # contrato mínimo de transação
    observability/                # logger slog (JSON)
    migrations/
        orders/                   # migrations do orders_db
        stock/                    # migrations do stock_db
    security/
```

---

## Regras de negócio implementadas

- Não permite criar pedidos sem cliente.
- Não permite criar pedidos sem itens.
- Não permite quantidade menor ou igual a zero.
- Verifica existência do cliente.
- Verifica existência do produto.
- Verifica estoque disponível.
- Reserva o estoque (via Saga) ao criar pedidos; compensa (libera) o que já foi reservado se algum item do pedido falhar.
- Devolve o estoque ao cancelar pedidos.
- Não permite cancelar pedidos pagos.
- Não permite pagar pedidos já cancelados.
- Não permite alterar pedidos já finalizados.
- Reserva e liberação de estoque são idempotentes por `(saga_id, product_id)` — reentrega de mensagem ou retry de rede não duplicam o efeito no estoque.
- Reservas de estoque órfãs (sem pedido correspondente) são liberadas automaticamente após um período configurável, por um job de reconciliação.

---

## Limitações conhecidas

- **Reconciliação por tempo**: reconciliação libera reservas órfãs com base só em quanto tempo elas estão paradas em `RESERVED` — ele não confirma com o order-service se existe de fato um pedido para cada `saga_id` antes de liberar. Teoricamente uma criação de pedido extremamente lenta poderia ter sua reserva liberada se ultrapassar o limiar. 
- **Liberação de estoque continua síncrona via HTTP** — só a reserva foi migrada para a Saga assíncrona; cancelamento/compensação ainda dependem do stock-service estar no ar no momento da chamada.
- Sem autenticação/autorização.

---

## Possíveis melhorias

Este projeto foi desenvolvido com fins de estudo e demonstração. Algumas melhorias que podem ser implementadas incluem:

- Autenticação (JWT)
- Autorização por perfis (RBAC)
- Documentação da API (OpenAPI/Swagger)
- Cache para consultas frequentes
- Paginação padronizada
- Filtros de busca
- CI/CD
- Reconciliação "saga-aware"
- Outros

---

## JSON

### Criar cliente (`POST /clientes`)

```json
{
  "name": "João Silva",
  "email": "joao.silva@email.com",
  "phone": "11999999999",
  "password": "123456"
}
```

---

### Criar produto (`POST /produtos`)

```json
{
  "name": "Notebook",
  "price": 5000.50,
  "stock": 10
}
```

---

### Criar pedido (`POST /pedidos`)

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

### Reservar/liberar estoque diretamente (`POST /produtos/{id}/reservar` ou `/liberar`)

`saga_id` é opcional — se omitido, o stock-service gera um novo (equivale a uma operação sempre aplicada, sem idempotência). O order-service sempre o envia.

```json
{
  "quantity": 3,
  "saga_id": "UUID_OPCIONAL"
}
```
