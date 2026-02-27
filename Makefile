.PHONY: help build run dev lint format docs-generate migrate-up migrate-down docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make build       	- Build the application"
	@echo "  make run         	- Run the application"
	@echo "  make dev         	- Run the application in development mode"
	@echo "  make lint        	- Run linter on the codebase"
	@echo "  make format      	- Format the code and re-arrange imports"
	@echo "  make docs-generate	- Generate docs API"
	@echo "  make migrate-up  	- Apply database migrations"
	@echo "  make migrate-down	- Rollback database migrations"
	@echo "  make docker-up   	- Run the application in a docker container"
	@echo "  make docker-down 	- Stop the docker container"

build:
	go build -o bin/app ./cmd/api

run:
	go run ./cmd/api

dev:
	go run ./cmd/api

lint: format
	golangci-lint run ./...

format:
	@gofmt -s -w .
	@goimports -w .

docs-generate:
	mkdir -p docs
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal --exclude .git,docs,docker,db

migrate-up:
	migrate -path db/migrations -database "postgresql://postgres:password@localhost:5433/ecommerce_shop?sslmode=disable" up

migrate-down:
	migrate -path db/migrations -database "postgresql://postgres:password@localhost:5433/ecommerce_shop?sslmode=disable" down

docker-up:
	docker compose -f docker/docker-compose.yml up -d

docker-down:
	docker compose -f docker/docker-compose.yml down