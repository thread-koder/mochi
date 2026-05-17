.PHONY: help \
	core-build core-test core-clean core-run core-dev core-fmt core-lint core-deps core-deps-update \
	docker-build docker-run \
	dev-env-setup dev-env-clean dev-env-status \
	test-workloads-setup test-workloads-clean

# Variables
CORE_DIR=core
CORE_BUILD_ENV=CGO_ENABLED=0
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
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-25s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Core (Go)
core-build: ## Build the core binary
	@echo "Building $(BINARY_NAME)..."
	@cd $(CORE_DIR) && $(CORE_BUILD_ENV) go build $(LDFLAGS) -o ../bin/$(BINARY_NAME) $(MAIN_PATH)

core-test: ## Run core tests (with race detector)
	@echo "Running core tests..."
	@cd $(CORE_DIR) && go test -v -race ./...

core-clean: ## Clean core build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@cd $(CORE_DIR) && go clean

core-run: core-build ## Build and run the core binary
	@./bin/$(BINARY_NAME)

core-dev: ## Run core in development mode
	@echo "Running core in development mode..."
	@cd $(CORE_DIR) && go run $(MAIN_PATH)

core-fmt: ## Format core code
	@echo "Formatting core code..."
	@cd $(CORE_DIR) && go fmt ./...

core-lint: ## Run go vet on core
	@echo "Running go vet on core..."
	@cd $(CORE_DIR) && go vet ./...

core-deps: ## Download and tidy core dependencies
	@echo "Downloading core dependencies..."
	@cd $(CORE_DIR) && go mod download && go mod tidy

core-deps-update: ## Update core dependencies
	@echo "Updating core dependencies..."
	@cd $(CORE_DIR) && go get -u ./... && go mod tidy

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(BINARY_NAME):$(VERSION) .

docker-run: docker-build ## Build and run Docker container
	@docker run --rm $(BINARY_NAME):$(VERSION)

# Development environment (minikube / Helm)
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
