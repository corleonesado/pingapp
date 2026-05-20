SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Short git SHA, or "dev" outside a repo.
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
IMAGE   ?= pingapp
TAG     ?= $(VERSION)
PORT    ?= 8080

.PHONY: help run test cover lint fmt tidy docker-build docker-run docker-stop compose-up compose-down

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-18s %s\n", $$1, $$2}'

run: ## Run the server locally with the current git SHA as version.
	PORT=$(PORT) go run -ldflags "-X main.version=$(VERSION)" ./cmd/server

test: ## Run unit tests with race detector and coverage.
	go test ./... -race -cover

cover: ## Generate an HTML coverage report at coverage.html.
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint (must be installed).
	golangci-lint run

fmt: ## Format Go code in place.
	gofmt -s -w .

tidy: ## Tidy go.mod / go.sum.
	go mod tidy

docker-build: ## Build the multi-stage Docker image, tagged with the git SHA and "dev".
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(TAG) -t $(IMAGE):dev .

docker-run: ## Run the image with port $(PORT) forwarded.
	docker run --rm --read-only -p $(PORT):8080 -e PORT=8080 --name $(IMAGE) $(IMAGE):dev

docker-stop: ## Stop any running container of the dev image.
	-docker stop $(IMAGE) 2>/dev/null

compose-up: ## Bring up the local stack via docker compose.
	docker compose up --build -d

compose-down: ## Tear down the local stack.
	docker compose down
