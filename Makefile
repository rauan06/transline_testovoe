.PHONY: help proto generate build clean test lint docker-up docker-down docker-build docker-logs

PROTO_DIR := api/proto
PROTO_FILE := $(PROTO_DIR)/customer.proto
GO_OUT_DIR := $(PROTO_DIR)

help: ## Показать справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

proto: ## Генерация Go кода из proto файлов (через buf)
	@echo "Генерация proto файлов через buf..."
	@buf generate

generate: proto 

build: proto ## Сборка всех сервисов
	@echo "Сборка customer-service..."
	@go build -o bin/customer-service ./cmd/customer-service
	@echo "Сборка shipment-service..."
	@go build -o bin/shipment-service ./cmd/shipment-service

run-customer: proto ## Запуск customer-service локально
	@echo "Запуск customer-service..."
	@go run ./cmd/customer-service

run-shipment: proto ## Запуск shipment-service локально
	@echo "Запуск shipment-service..."
	@go run ./cmd/shipment-service

test: ## Запуск тестов
	@echo "Запуск тестов..."
	@go test -v ./...

test-coverage: ## Запуск тестов с покрытием
	@echo "Запуск тестов с покрытием..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Очистка сгенерированных файлов и бинарников
	@echo "Очистка..."
	@rm -rf bin/
	@rm -rf $(PROTO_DIR)/*.pb.go
	@rm -rf $(PROTO_DIR)/*_grpc.pb.go
	@rm -f coverage.out coverage.html

docker-build: ## Сборка Docker образов
	@echo "Сборка Docker образов..."
	@docker-compose build

run: ## Запуск всех сервисов через docker-compose
	@echo "Запуск сервисов..."
	@docker-compose up -d

stop: ## Остановка всех сервисов
	@echo "Остановка сервисов..."
	@docker-compose down

docker-logs: ## Просмотр логов всех сервисов
	@docker-compose logs -f

docker-logs-customer: ## Просмотр логов customer-service
	@docker-compose logs -f customer-service

docker-logs-shipment: ## Просмотр логов shipment-service
	@docker-compose logs -f shipment-service

docker-logs-envoy: ## Просмотр логов envoy
	@docker-compose logs -f envoy

install-deps: ## Установка зависимостей Go
	@echo "Установка зависимостей..."
	@go mod download
	@go mod tidy

install-tools: ## Установка необходимых инструментов
	@echo "Установка buf..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "Установка protoc-gen-go..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0
	@echo "Установка protoc-gen-go-grpc..."
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

setup: install-deps install-tools ## Полная настройка проекта

.DEFAULT_GOAL := help

