MIGRATIONS_DIR = migrations
MFILE_NAME ?= create_payment_table
DB_URL = postgres://payment_user:payment_pass@localhost:5433/payment_db?sslmode=disable

.PHONY: migrate migup migdown

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