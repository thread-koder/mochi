.PHONY: help \
	core-build core-clean core-run core-dev core-fmt core-lint core-deps core-deps-update \
	agent-build agent-clean agent-run agent-dev agent-fmt agent-lint agent-generate agent-deps agent-deps-update \
	web-build web-clean web-dev web-prepare web-lint web-deps \
	dev-env-setup dev-env-cleanup \
	agent-setup agent-cleanup \
	test-workloads-setup test-workloads-cleanup

# Variables
WEB_DIR=web
PNPM=corepack pnpm
CORE_DIR=core
AGENT_DIR=agent
CORE_BINARY_NAME=mochi
AGENT_BINARY_NAME=mochi-agent
CORE_MAIN_PATH=./cmd/mochi
AGENT_MAIN_PATH=./cmd/agent
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_BUILD_ENV=CGO_ENABLED=0
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-25s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Core (Go)
core-build: ## Build the core binary
	@echo "Building $(CORE_BINARY_NAME)..."
	@cd $(CORE_DIR) && $(GO_BUILD_ENV) go build $(LDFLAGS) -o ../bin/$(CORE_BINARY_NAME) $(CORE_MAIN_PATH)

core-clean: ## Clean core build artifacts
	@echo "Cleaning..."
	@rm -f bin/$(CORE_BINARY_NAME)
	@cd $(CORE_DIR) && go clean

core-run: core-build ## Build and run the core binary
	@./bin/$(CORE_BINARY_NAME)

core-dev: ## Run core in development mode
	@echo "Running core in development mode..."
	@cd $(CORE_DIR) && go run $(CORE_MAIN_PATH)

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

# Agent (Go)
agent-build: ## Build the agent binary
	@echo "Building $(AGENT_BINARY_NAME)..."
	@cd $(AGENT_DIR) && $(GO_BUILD_ENV) go build $(LDFLAGS) -o ../bin/$(AGENT_BINARY_NAME) $(AGENT_MAIN_PATH)

agent-clean: ## Clean agent build artifacts
	@echo "Cleaning..."
	@rm -f bin/$(AGENT_BINARY_NAME)
	@cd $(AGENT_DIR) && go clean

agent-run: agent-build ## Build and run the agent binary
	@./bin/$(AGENT_BINARY_NAME)

agent-dev: ## Run agent in development mode
	@echo "Running agent in development mode..."
	@cd $(AGENT_DIR) && go run $(AGENT_MAIN_PATH)

agent-fmt: ## Format agent code
	@echo "Formatting agent code..."
	@cd $(AGENT_DIR) && go fmt ./...

agent-lint: ## Run go vet on agent
	@echo "Running go vet on agent..."
	@cd $(AGENT_DIR) && go vet ./...

# Requires clang and bpf/vmlinux.h (see agent/internal/collection/ebpf/generate.go).
agent-generate: ## Regenerate agent eBPF Go bindings (bpf2go)
	@echo "Generating agent eBPF bindings..."
	@cd $(AGENT_DIR) && go generate ./internal/collection/ebpf

agent-deps: ## Download and tidy agent dependencies
	@echo "Downloading agent dependencies..."
	@cd $(AGENT_DIR) && go mod download && go mod tidy

agent-deps-update: ## Update agent dependencies
	@echo "Updating agent dependencies..."
	@cd $(AGENT_DIR) && go get -u ./... && go mod tidy

# Web (Nuxt)
web-build: ## Build Nuxt for production
	@echo "Building web..."
	@cd $(WEB_DIR) && $(PNPM) build

web-clean: ## Remove Nuxt build artifacts
	@echo "Cleaning web build artifacts..."
	@rm -rf $(WEB_DIR)/.output $(WEB_DIR)/.nitro $(WEB_DIR)/.cache $(WEB_DIR)/.data

web-dev: ## Run Nuxt dev server
	@echo "Running web dev server..."
	@cd $(WEB_DIR) && $(PNPM) dev

web-prepare: ## Generate Nuxt types and module stubs
	@echo "Preparing web..."
	@cd $(WEB_DIR) && $(PNPM) exec nuxt prepare

web-lint: ## Lint web code
	@echo "Linting web..."
	@cd $(WEB_DIR) && $(PNPM) lint

web-deps: ## Install web dependencies
	@echo "Installing web dependencies..."
	@cd $(WEB_DIR) && $(PNPM) install

# Development environment (minikube / Helm)
dev-env-setup: ## Set up PostgreSQL, Prometheus, and Redis in minikube using Helm
	@chmod +x scripts/setup-dev-env.sh
	@./scripts/setup-dev-env.sh

dev-env-cleanup: ## Remove PostgreSQL, Prometheus, and Redis from minikube
	@chmod +x scripts/cleanup-dev-env.sh
	@./scripts/cleanup-dev-env.sh

agent-setup: ## Build mochi-agent:dev into minikube and apply agent/deploy
	@chmod +x scripts/setup-agent.sh
	@./scripts/setup-agent.sh

agent-cleanup: ## Remove mochi-agent from minikube
	@chmod +x scripts/cleanup-agent.sh
	@./scripts/cleanup-agent.sh

test-workloads-setup: ## Set up test workloads in minikube
	@chmod +x scripts/setup-test-workloads.sh
	@./scripts/setup-test-workloads.sh

test-workloads-cleanup: ## Remove test workloads from minikube
	@chmod +x scripts/cleanup-test-workloads.sh
	@./scripts/cleanup-test-workloads.sh
