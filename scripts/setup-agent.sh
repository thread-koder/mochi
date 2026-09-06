#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/agent/deploy"
IMAGE="mochi-agent:dev"

echo -e "${GREEN}Setting up mochi-agent in minikube${NC}\n"

# Check if minikube is running
if ! minikube status &>/dev/null; then
    echo -e "${RED}Minikube is not running. Please start minikube first.${NC}"
    exit 1
fi

if ! command -v docker &>/dev/null; then
    echo -e "${RED}Docker is not installed or not on PATH.${NC}"
    exit 1
fi

# Namespace is owned by dev-env-setup
if ! kubectl get namespace "${NAMESPACE}" &>/dev/null; then
    echo -e "${RED}Namespace '${NAMESPACE}' does not exist. Run 'make dev-env-setup' first.${NC}"
    exit 1
fi

echo -e "${GREEN}Building ${IMAGE}...${NC}"
docker build -f "${REPO_ROOT}/agent/Dockerfile" -t "${IMAGE}" "${REPO_ROOT}"

echo -e "${GREEN}Loading image into minikube...${NC}"
minikube image load "${IMAGE}"

echo -e "${GREEN}Deploying mochi-agent...${NC}"
kubectl apply -f "${DEPLOY_DIR}/rbac.yaml"
kubectl apply -f "${DEPLOY_DIR}/daemonset.yaml"
kubectl apply -f "${DEPLOY_DIR}/podmonitor.yaml"

echo -e "${GREEN}Waiting for mochi-agent to be ready...${NC}"
kubectl -n "${NAMESPACE}" rollout status daemonset/mochi-agent --timeout=60s

echo -e "${GREEN}mochi-agent deployed successfully!${NC}"
echo -e "${GREEN}Tip: Use 'make agent-cleanup' to remove mochi-agent${NC}"
