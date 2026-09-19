SHELL := /bin/sh
COMPOSE := docker compose

.PHONY: help dev down logs build test vet fmt seed scan migrate

help:
	@grep -E '^[a-z-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-10s %s\n", $$1, $$2}'

dev: ## Start all services
	$(COMPOSE) up -d --build

down: ## Stop all services
	$(COMPOSE) down

logs: ## Follow logs
	$(COMPOSE) logs -f api worker

build: ## Build the backend
	cd backend && go build ./...

test: ## Run the tests
	cd backend && go test ./...

vet: ## Static analysis
	cd backend && go vet ./...

fmt: ## Check formatting
	@cd backend && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

seed: ## Load the agency list
	cd backend && go run ./cmd/seed

scan: ## Check a single URL: make scan URL=https://www.bund.de
	cd backend && go run ./cmd/scan -url "$(URL)"
