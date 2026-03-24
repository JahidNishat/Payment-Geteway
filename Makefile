PROTO_DIR = proto
PAYMENT_PROTO = $(PROTO_DIR)/payment/v1/payment.proto
GEN_DIR = gen/go
MIGRATIONS_DIR = services/payment/migrations
MFILE_NAME ?= create_payment_table
DB_URL = postgres://postgres:postgres@localhost:5432/payment_gateway?sslmode=disable

.PHONY: proto-gen clean migrate migup migdown

proto-gen:
	@echo "Generating Go code from proto..."
	@mkdir -p $(GEN_DIR)
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PAYMENT_PROTO)
	@echo "Done!"

clean:
	@echo "Cleaning generated files..."
	@rm -rf $(GEN_DIR)
	@echo "Done!"

migrate:
	@echo "Creating database migration: $(MFILE_NAME)"
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(MFILE_NAME)
	@echo "Done!"

migup:
	@echo "Applying database migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up
	@echo "Done!"

migdown:
	@echo "Rolling back database migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down
	@echo "Done!"