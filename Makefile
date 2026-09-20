SHELL := /bin/sh
COMPOSE := docker compose

.PHONY: help dev dev-telemetry down logs build test test-db test-web vet fmt seed scan web

TEST_DATABASE_URL ?= postgres://behoerdenbarriere:behoerdenbarriere@localhost:5432/behoerdenbarriere?sslmode=disable

help:
	@grep -E '^[a-z-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-10s %s\n", $$1, $$2}'

dev: ## Start all services
	$(COMPOSE) up -d --build

dev-telemetry: ## Start all services plus the OTLP collector
	$(COMPOSE) --profile telemetry up -d --build

down: ## Stop all services
	$(COMPOSE) --profile telemetry down

logs: ## Follow logs
	$(COMPOSE) logs -f api worker

build: ## Build the backend
	cd backend && go build ./...

test: ## Run the tests (database and browser tests are skipped)
	cd backend && go test -short ./...

test-db: ## Run all tests, including database and browser
	$(COMPOSE) up -d postgres
	cd backend && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./...

test-web: ## Run the frontend tests
	cd frontend && npm test

e2e: ## Run the end-to-end tests (desktop, phone, 320 px)
	cd frontend && npm run e2e

web: ## Start the frontend in development mode
	cd frontend && npm run dev

vet: ## Static analysis
	cd backend && go vet ./...

fmt: ## Check formatting
	@cd backend && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

seed: ## Load the agency list
	cd backend && go run ./cmd/seed

scan: ## Check a single URL: make scan URL=https://www.bund.de
	cd backend && go run ./cmd/scan -url "$(URL)"
