#!/bin/bash
echo "Setting up payment-gateway project..."

# Proto definitions
mkdir -p proto/payment/v1

# Payment service
mkdir -p services/payment/cmd/server
mkdir -p services/payment/internal/{config,handler,service,repository,model,errors,events,processor}
mkdir -p services/payment/migrations

# Notification service  
mkdir -p services/notification/cmd/worker
mkdir -p services/notification/internal/{config,webhook,consumer,repository,model}
mkdir -p services/notification/migrations

# CI
mkdir -p .github/workflows

# Create files
touch docker-compose.yaml
touch Makefile
touch .env.example
touch README.md
touch .gitignore

echo "Done!"