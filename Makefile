SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Short git SHA, or "dev" outside a repo.
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
IMAGE   ?= pingapp
TAG     ?= $(VERSION)
PORT    ?= 8080

CHART       ?= charts/pingapp
RELEASE     ?= pingapp
NAMESPACE   ?= default

.PHONY: help run test cover lint fmt tidy docker-build docker-run docker-stop compose-up compose-down \
        minikube-up minikube-down minikube-load helm-lint helm-template-dev helm-template-prod \
        deploy-dev deploy-prod uninstall rollback history status

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

# ---- Kubernetes (Track B: minikube) ----------------------------------------

minikube-up: ## Start a minikube cluster with ingress + metrics-server addons.
	minikube start --driver=docker --addons=ingress --addons=metrics-server --memory=4g --cpus=2

minikube-down: ## Stop the minikube cluster (does not delete the profile).
	minikube stop

minikube-load: docker-build ## Build the image and load it into minikube's docker daemon.
	minikube image load $(IMAGE):dev
	minikube image load $(IMAGE):$(TAG)

helm-lint: ## Lint the chart with default, dev, and prod values.
	helm lint $(CHART)
	helm lint $(CHART) -f $(CHART)/values-dev.yaml
	helm lint $(CHART) -f $(CHART)/values-prod.yaml

helm-template-dev: ## Render the chart with dev values to stdout (for review).
	helm template $(RELEASE) $(CHART) -f $(CHART)/values-dev.yaml

helm-template-prod: ## Render the chart with prod values to stdout (for review).
	helm template $(RELEASE) $(CHART) -f $(CHART)/values-prod.yaml

deploy-dev: minikube-load ## Build, load, and helm upgrade --install with dev values.
	helm upgrade --install $(RELEASE) $(CHART) \
		-n $(NAMESPACE) --create-namespace \
		-f $(CHART)/values-dev.yaml \
		--set image.tag=$(TAG) \
		--wait --timeout 2m

deploy-prod: minikube-load ## Build, load, and helm upgrade --install with prod values.
	helm upgrade --install $(RELEASE) $(CHART) \
		-n $(NAMESPACE) --create-namespace \
		-f $(CHART)/values-prod.yaml \
		--set image.tag=$(TAG) \
		--wait --timeout 2m

uninstall: ## Remove the pingapp release from the cluster.
	helm uninstall $(RELEASE) -n $(NAMESPACE)

rollback: ## Roll back to the previous release revision (helm rollback $(RELEASE) 0).
	helm rollback $(RELEASE) 0 -n $(NAMESPACE)
	kubectl rollout status deployment/$(RELEASE) -n $(NAMESPACE)

history: ## Show helm release history.
	helm history $(RELEASE) -n $(NAMESPACE)

status: ## Show pods, service, ingress, and rollout status.
	kubectl get pods,svc,ingress -l app.kubernetes.io/instance=$(RELEASE) -n $(NAMESPACE)
	@echo "---"
	kubectl rollout status deployment/$(RELEASE) -n $(NAMESPACE) --timeout=10s || true
