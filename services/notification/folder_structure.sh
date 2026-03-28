#!/usr/bin/env bash
set -euo pipefail

echo "🚀 Bootstrapping the Notification Worker Service..."

# 1. Create all directories
mkdir -p cmd/worker
mkdir -p internal/config
mkdir -p internal/consumer
mkdir -p internal/model
mkdir -p internal/repository
mkdir -p internal/webhook
mkdir -p migrations

# 2. Create files and inject the package names
echo "package main" > cmd/worker/main.go
echo "package config" > internal/config/config.go
echo "package consumer" > internal/consumer/consumer.go
echo "package model" > internal/model/webhook.go
echo "package repository" > internal/repository/webhook_repository.go
echo "package webhook" > internal/webhook/sender.go

# Create the empty migration file
touch migrations/000001_create_webhook_tables.up.sql

# 3. Initialize the Go module (if it doesn't already exist)
if [ ! -f go.mod ]; then
    echo "📦 Initializing Go module..."
    go mod init github.com/JahidNishat/payment-gateway/services/notification
fi

echo "✅ Structure created successfully!"
echo "📂 Run 'tree' to view your new workspace."