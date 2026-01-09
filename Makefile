.PHONY: help build test clean run lint fmt docker-build docker-run deps deps-update dev dev-env-setup dev-env-clean dev-env-status test-workloads-setup test-workloads-clean

# Variables
BINARY_NAME=mochi
MAIN_PATH=./cmd/mochi
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

run: build ## Build and run the application
	@./bin/$(BINARY_NAME)

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(BINARY_NAME):$(VERSION) .

docker-run: docker-build ## Build and run Docker container
	@docker run --rm $(BINARY_NAME):$(VERSION)

dev: ## Run in development mode
	@echo "Running in development mode..."
	@go run $(MAIN_PATH)

dev-env-setup: ## Set up PostgreSQL, Prometheus, and Redis in minikube using Helm
	@chmod +x scripts/setup-dev.sh
	@./scripts/setup-dev.sh

dev-env-clean: ## Remove PostgreSQL, Prometheus, and Redis from minikube
	@chmod +x scripts/cleanup-dev.sh
	@./scripts/cleanup-dev.sh

dev-env-status: ## Check status of development environment
	@chmod +x scripts/status-dev.sh
	@./scripts/status-dev.sh

test-workloads-setup: ## Set up test workloads (Deployment, DaemonSet, Standalone Pod) in minikube
	@chmod +x scripts/setup-test-workloads.sh
	@./scripts/setup-test-workloads.sh

test-workloads-clean: ## Remove test workloads from minikube
	@chmod +x scripts/cleanup-test-workloads.sh
	@./scripts/cleanup-test-workloads.sh
