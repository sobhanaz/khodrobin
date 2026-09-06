.PHONY: help up down logs test lint build fmt

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

up: ## Start the stack
	docker compose up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail all logs
	docker compose logs -f --tail=100

test: ## Run every test suite
	cd api && go test -race ./...

lint: ## Vet and format-check
	cd api && go vet ./... && test -z "$$(gofmt -l .)"

fmt: ## Format
	cd api && gofmt -w .

build: ## Build the API binary
	cd api && go build -o bin/api ./cmd/api
