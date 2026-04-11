FROM golang:1.25-alpine3.23 AS builder

WORKDIR /app

# Copy the gen module first (shared dependency)
COPY gen/go/ gen/go/

# Copy the payment service module
COPY services/payment/ services/payment/

# Download deps from the payment service directory
WORKDIR /app/services/payment
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/payment-service ./cmd/server/main.go

# Runtime stage
FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/payment-service .

COPY --from=builder /app/services/payment/migrations ./migrations

EXPOSE 50051

CMD [ "./payment-service" ]