# Payment Gateway
Event-driven payment processing system built with Go, gRPC, and NATS JetStream. Handles payment creation, processing, refunds, and merchant webhook notifications with idempotency guarantees and full audit trails.

---

## Architectural Diagram
```text
┌──────────┐         gRPC             ┌──────────────────┐
│  Client  │ ───────────────────────► │  Payment Service │
└──────────┘                          │                  │
                                      │  • Validation    │
                                      │  • Idempotency   │
                                      │  • Processing    │
                                      └──────┬───┬───────┘
                                             │   │
                              ┌──────────────┘   └──────────────┐
                              │                                 │
                              ▼                                 ▼
                    ┌──────────────┐                   ┌──────────────────┐
                    │  PostgreSQL  │                   │   NATS JetStream │
                    │              │                   │                  │
                    │  • Payments  │                   │  • Payment events│
                    │  • Refunds   │                   │  • Persistent    │
                    │  • Events    │                   │  • Queue groups  │
                    └──────────────┘                   └────────┬─────────┘
                                                                │
                                                                ▼
                                                   ┌─────────────────────┐
                                                   │Notification Service │
                                                   │                     │
                                                   │ • Webhook delivery  │
                                                   │ • Retry w/ backoff  │
                                                   │ • Attempt logging   │
                                                   │ • Idempotent        │
                                                   └──────────┬──────────┘
                                                              │
                                                              ▼
                                                    ┌──────────────────┐
                                                    │  Merchant Server │
                                                    │  (Webhook POST)  │
                                                    └──────────────────┘
```
---

## Features

### Payment Processing
- Synchronous payment processing with simulated bank integration
- Idempotent payment creation (duplicate requests return same result)
- Status state machine: PENDING → PROCESSING → COMPLETED/FAILED
- Full audit trail of every status transition

### Refund Handling
- Partial and full refund support
- Refund amount validation against original payment
- Processor integration for refund execution

### Event-Driven Notifications
- Asynchronous event publishing via NATS JetStream
- Durable consumers with queue groups (scalable, no duplicates)
- Webhook delivery to merchant endpoints
- Exponential backoff retry (3 attempts)
- Idempotent delivery (database deduplication)
- Full webhook attempt logging

### Infrastructure
- gRPC API with Protocol Buffers
- PostgreSQL with database transactions and row-level locking
- Docker multi-stage builds (~15MB images)
- Docker Compose full-stack orchestration with health checks
- Structured JSON logging
- Graceful shutdown
- gRPC interceptors (logging, panic recovery)
- Request-level timeouts

---

## Tech Stack

| Technology | Purpose |
|-----------|---------|
| Go 1.25 | Primary language |
| gRPC + Protobuf | Service API |
| PostgreSQL 16 | Primary database |
| NATS JetStream | Async event streaming |
| Docker + Compose | Containerization and orchestration |
| golang-migrate | Database migrations |
| sqlx | Database access |
| slog | Structured logging |
| GitHub Actions | CI pipeline |

---

## Project Structure

```text
.
├── proto/                          # Protobuf definitions
├── gen/                            # Generated gRPC code
├── services/
│   ├── payment/
│   │   ├── cmd/server/             # Entry point
│   │   ├── internal/
│   │   │   ├── config/             # Configuration
│   │   │   ├── handler/            # gRPC handlers + error mapping
│   │   │   ├── service/            # Business logic + tests
│   │   │   ├── repository/         # Database access
│   │   │   ├── processor/          # Payment processor (simulated)
│   │   │   ├── events/             # NATS publisher
│   │   │   ├── interceptor/        # gRPC middleware
│   │   │   ├── errors/             # Custom error types
│   │   │   ├── model/              # Domain models
│   │   │   └── validator/          # Request validation
│   │   └── migrations/             # SQL migrations
│   └── notification/
│       ├── cmd/worker/             # Entry point
│       ├── internal/
│       │   ├── config/             # Configuration
│       │   ├── consumer/           # NATS JetStream consumer
│       │   ├── webhook/            # HTTP webhook sender with retry
│       │   ├── repository/         # Webhook attempt logging
│       │   └── model/              # Domain models
│       └── migrations/             # SQL migrations
└── docker-compose.yaml             # Full stack orchestration
```

---

## How to run

- Clone the repo
- Have to install Docker and Docker Compose
- Use Docker Compose to run both services with PostgreSQL and NATS
  - To Up Docker Compose

    ```bash
    docker-compose up --build
    ```

  - To Down Docker Compose

    ```bash
    docker-compose down -v
    ```

---

## Enviroment Variables

### Postgresql:
- POSTGRES_USER (Required)
- POSTGRES_PASSWORD (Required)
- POSTGRES_DB (Required)

### Notification Service:
- DB_URL (Required)
- NATS_URL (Default: nats://localhost:4222)

### Payment Service:
- DB_URL (Required)
- NATS_URL (Default: nats://localhost:4222)
- GRPC_PORT "Default: 50051"

---

## API Example:

### Create Payment:
```curl
grpcurl -plaintext -d '{
  "idempotency_key": "docker-test-001",
  "merchant_id": "merchant-1",
  "amount": 50000,
  "currency": "BDT"
}' localhost:50051 payment.v1.PaymentService/CreatePayment
```

### Get Payment:
```curl
grpcurl -plaintext -d '{
  "payment_id": "9f13b383-a7ab-40e0-a0a6-265a2f12c5ae"
}' localhost:50051 payment.v1.PaymentService/GetPayment
```

### List Payment:
```curl
grpcurl -plaintext -d '{
  "merchant_id": "merchant-1",
  "page": 1,
  "limit": 10
}' localhost:50051 payment.v1.PaymentService/ListPayments
```

### Refund Payment:
```curl
grpcurl -plaintext -d '{
  "payment_id": "2e12e7ce-13b3-42f6-bb6f-1430f7bcddf7",
  "amount": 20000,
  "reason": "Customer requested partial refund"
}' localhost:50051 payment.v1.PaymentService/RefundPayment
```