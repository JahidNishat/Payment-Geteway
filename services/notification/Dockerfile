FROM golang:1.25-alpine3.23 AS builder

WORKDIR /app

COPY gen/go/ gen/go/
COPY services/notification/ services/notification/

WORKDIR /app/services/notification
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/notification-service ./cmd/worker/main.go

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/notification-service .

CMD [ "./notification-service" ]