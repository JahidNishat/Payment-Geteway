# Payment Gateway — System Design

## 1. Overview
An event driven architecture where a payment processing system is build, implemented with Go, gRPC, Postgresql, NATS JetStream. It can create a payment, get payment list by merchant id, supports partial/full refunds, get each payment refund status, also it gurantees idempotency so that no duplicate transaction is not possible and has full log or audit trails.

## 2. Requirements
### Functional
These define specific behavoirs, data processing, and interactions.
- **Payment Creation:** Client can create a payment using idempotency key, amount.
- **Payment Details:** Client can check payment details with payment id
- **Payment List:** Client can get paginated payment list for a merchant
- **Refund Payment:** Client can initiate partial or full refund of a payment

### Non-Functional Requirements
- **Consistency:** Payment state must be strongly consistent. A payment cannot be both COMPLETED and FAILED.
- **Durability:** Once a payment is marked COMPLETED, that state must survive system crashes (PostgreSQL ACID guarantees this).
- **Availability:** System should handle service restarts without losing in-flight payments or events (NATS JetStream persistence).
- **Latency:** Payment creation should complete within 2 seconds (including processor simulation of 500ms).
- **Idempotency:** Duplicate requests with same idempotency key must return identical results without side effects.

## 3. API Design
- **Create Payment:**

    _Request_
    ```rpc
    grpcurl -plaintext -d '{
    "idempotency_key": "docker-test-001",
    "merchant_id": "merchant-1",
    "amount": 50000,
    "currency": "BDT"
    }' localhost:50051 payment.v1.PaymentService/CreatePayment
    ```
    _Response_
    ```rpc
    {
    "paymentId": "daf90428-538c-413d-83b5-1c94970e8280",
    "status": "PAYMENT_STATUS_COMPLETED",
    "amount": "50000",
    "currency": "BDT",
    "createdAt": "2026-04-01T14:58:15.581326Z"
    }
    ```
- **Get Payment:**

    _Request_
    ```curl
    grpcurl -plaintext -d '{
    "payment_id": "daf90428-538c-413d-83b5-1c94970e8280"
    }' localhost:50051 payment.v1.PaymentService/GetPayment
    ```
    _Response_
    ```curl
    {
    "paymentId": "daf90428-538c-413d-83b5-1c94970e8280",
    "merchantId": "merchant-1",
    "amount": "50000",
    "currency": "BDT",
    "status": "PAYMENT_STATUS_COMPLETED",
    "createdAt": "2026-04-01T14:58:15.581326Z",
    "updatedAt": "2026-04-01T14:58:16.089405Z"
    }
    ```
- **List Payment By Merchant ID:**

    _Request_
    ```curl
    grpcurl -plaintext -d '{
    "merchant_id": "merchant-1",
    "page": 1,
    "limit": 10
    }' localhost:50051 payment.v1.PaymentService/ListPayments
    ```
    _Response_
    ```curl
    {
    "payments": [
        {
        "paymentId": "daf90428-538c-413d-83b5-1c94970e8280",
        "merchantId": "merchant-1",
        "amount": "50000",
        "currency": "BDT",
        "status": "PAYMENT_STATUS_COMPLETED",
        "createdAt": "2026-04-01T14:58:15.581326Z"
        }
    ],
    "totalCount": 1,
    "page": 1,
    "limit": 10
    }
    ```
- **Refund Payment:**

    _Request_
    ```curl
    grpcurl -plaintext -d '{
    "payment_id": "2e12e7ce-13b3-42f6-bb6f-1430f7bcddf7",
    "amount": 20000,
    "reason": "Customer requested partial refund"
    }' localhost:50051 payment.v1.PaymentService/RefundPayment
    ```
    _Response_
    ```curl
    {
    "refundId": "e124f820-894f-489a-9d56-b581d4b33181",
    "paymentId": "2e12e7ce-13b3-42f6-bb6f-1430f7bcddf7",
    "amount": "20000",
    "reason": "Customer requested partial refund",
    "status": "REFUND_STATUS_COMPLETED"
    }
    ```
## 4. Data Model
### Payments Table
Stores every payment attempt. Uses `amount` as BIGINT (cents) to avoid floating-point precision errors. Status is constrained to valid values via CHECK constraint.

**Key Index:** `(merchant_id, idempotency_key)` UNIQUE
- Enforces idempotency at the database level
- Prevents race conditions when two identical requests arrive simultaneously

### Payment Events Table  
Audit trail recording every status transition. Enables debugging and compliance. Uses ON DELETE RESTRICT to prevent orphaned events.

**Why separate table?** 
Embedding status history in the payments table would require array columns or JSON, making queries harder. A separate table gives clean relational queries and chronological ordering.

### Refunds Table
Tracks partial and full refunds. Amount is validated against total payment amount to prevent over-refunding.

### Webhook Attempts Table (Notification Service)
Records every webhook delivery attempt. Enables debugging merchant integration issues and provides delivery guarantees 
through idempotent processing.

## 5. Architecture
- There are two services: `Notification Service`, `Payment Service`
- Client's Request land on `Payment Service`, after validate the request it sends to Proccessor, there two cases happens:
    - _Fail:_ If the request failed, then the payment service send fail reponse back to client.
    - _Success:_ If the request success, then payment sevice publish an event through nats jetstream and give success Reponse to the client.
- Notification subscribe to a subject and queue to the nats server, when a event published from the payment service, notification subscriber recieves it and start to process.
- Nats message handler check the request, and fetch the url according to the merchant.
- Then a webhook is called:
    - If failed, then theres a max retry limit, it retries until then and records the result.
    - If success, then records the result to database.
- Give acknowledgement to the NATS JetStream

## 6. Key Design Decisions
### 6.1 Why gRPC?
**Problem:**
In RESTful Api request and response usually use JSON payload. In JSON payload, we send the field name every time. So in machine level, it has to convert to byte format and receiving side also have to do the same, So data serialization and deserialization takes much time so this. And also same field name is used so extra data and bandwidth is also consumed.
**Solution:** ...
In gRPC, we create a contract like in 1st field we will send name, in 2nd field we will send email etc etc, so we don't send the full field name every time instead we send then in serial, and in binary form, though it is closed to machine language, it takes way more less time for serializing and deserializing, also take way more less data and bandwidth to send, so request response is faster than the RESTful api.
**Tradeoff:** ...
Though this binary form is faster than JSON, the JSON is more human readable, but the binary is not. gRPC natively runs HTTP/2 which is not supported by all the web browser, so in web based application must have to use the proxy. Though gRPC relies on the strict contact(.proto) both server and client side, it makes it more tightly coupled. And gRPC has a steeper learning curve.

### 6.2 Why Synchronous Payment Processing?

**Problem:**
Merchants need to know immediately whether a payment succeeded or failed. If processing is asynchronous, the merchant must poll for status or wait for a webhook — adding complexity on both sides.

**Solution:**
Payment processing happens synchronously within the CreatePayment request. The client sends a request and receives the final status (COMPLETED or FAILED) 
in the same response.

**Why:**
- Simpler client integration (no polling, no callback handling)
- Immediate feedback for the merchant
- Easier error handling (error returned directly)

**Tradeoff:**
- Client blocks for ~500ms during processing
- If processor is slow or hangs, client waits (mitigated by 30s timeout)
- Cannot easily distribute processing across workers
- In production with high volume, async processing with webhooks would be more scalable

### 6.3 Why NATS JetStream
**Problem:**
Payment service needs to notify other services without blocking the payment response.

**Solution:**
Publish state transition events asynchronously to a NATS JetStream topic, allowing downstream services to consume them independently at their own pace.

**Why not RabbitMQ/Kafka:**
Kafka introduces heavy operational complexity (JVM tuning, KRaft/Zookeeper management) which is often overkill for internal pub/sub eventing. RabbitMQ is feature-rich but can become a bottleneck under high-throughput persistent loads. NATS JetStream provides a highly performant, lightweight binary (written in Go) that is trivial to cluster, offers at-least-once delivery, and handles persistent stream replays efficiently with a fraction of the infrastructure overhead.

**Tradeoff:**
NATS JetStream has a smaller third-party ecosystem compared to Kafka. If the system later requires deep, out-of-the-box integrations with big data data warehouses or massive stream-processing frameworks (like Kafka Connect or Flink), NATS will require more custom integration work.

### 6.4 Why Separate Notification Service
**Problem:**
Could have sent webhooks directly from payment service.

**Solution:**
Extract all outbound communication into a dedicated Notification Service that subscribes to the NATS JetStream and manages webhook delivery.

**Why:**
Fault isolation and resource protection. Outbound webhooks are inherently unpredictable—third-party endpoints can be slow, rate-limit you, or go offline entirely. If the core payment service handled this, a slow webhook receiver could tie up active goroutines, drain HTTP connection pools, and eventually degrade the latency and throughput of the actual payment processing pipeline. A separate service isolates this backpressure and handles exponential backoff retries safely.

**Tradeoff:**
Increased distributed system complexity. It introduces another microservice that needs to be deployed, monitored, and scaled. It also adds a slight end-to-end latency bump to the notification delivery due to the network hop through the message broker.

## 7. Failure Handling
- **Payment Proccessor Error:** When payment procees return errors like network error or proccessor service internal error, we saved the payment as failed and the error. When there is response, we check if the response has error or not, if there is error then we saved the payment as failed and save the reason.
- **NATS:** When the NATS is down while starting the server we through error and down the server. But if there is error while running the server, we log as `CRITICAL` but doesn't off the server. Cause payment can be going on there.
- **Notification Service:** If the notification service is crashes then the events will be storing inside the `NATS` until it starts consuming again. And it will continue consuming from the last unacknowledgement message.
- **Merchant Webhook:** If the merchant webhook returns 500, we will retry until the max retry and each attempt we save the response.
- **Database:** If the database is down while starting the server, we show error and exist.
## 8. Scaling Considerations
- **Database connection pooling:** With multiple payment service instances, each opening connections, the database can hit its max_connections limit. A connection pooler like PgBouncer would sit between services and the database.

- **NATS clustering:** Single NATS server is a single point of failure. A 3-node NATS cluster provides fault tolerance.

- **Caching:** Idempotency checks currently query PostgreSQL on every request. Redis could cache recent idempotency keys for faster lookups.

- **Database partitioning:** The payments table grows indefinitely. Range partitioning by created_at (monthly) would keep query performance stable.

- **Async payment processing:** Synchronous processing blocks the client. At 10K+/s, async processing with webhook callbacks would reduce latency and allow horizontal scaling of processors.

## 9. What I Would Do Differently
- **Outbox pattern** for event publishing — currently if the app crashes after DB commit but before NATS publish, the event is lost
- **Merchant management service** with registered webhook URLs stored in database instead of hardcoded map
- **JWT authentication** for API access with merchant API keys
- **Observability** — Prometheus metrics, distributed tracing, alerting on payment failure rates
- **Database read replicas** for ListPayments queries to reduce load on the primary