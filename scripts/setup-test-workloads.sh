#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKLOADS_FILE="${SCRIPT_DIR}/test-workloads.yaml"

echo -e "${GREEN}Setting up test workloads in minikube${NC}\n"

# Check if minikube is running
if ! minikube status &>/dev/null; then
    echo -e "${RED}Minikube is not running. Please start minikube first.${NC}"
    exit 1
fi

# Check if namespace exists
if ! kubectl get namespace "${NAMESPACE}" &>/dev/null; then
    echo -e "${RED}Namespace '${NAMESPACE}' does not exist. Run 'make dev-env-setup' first.${NC}"
    exit 1
fi

# Deploy test workloads
echo -e "${GREEN}Deploying test workloads...${NC}"
kubectl apply -f "${WORKLOADS_FILE}"

echo -e "${GREEN}Waiting for workloads to be ready...${NC}"
kubectl -n "${NAMESPACE}" rollout status deployment/test-backend --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deployment/test-frontend --timeout=120s
kubectl -n "${NAMESPACE}" rollout status statefulset/test-cache --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deployment/test-worker --timeout=120s
kubectl -n "${NAMESPACE}" rollout status daemonset/test-daemonset --timeout=120s
kubectl -n "${NAMESPACE}" wait pod/test-standalone-pod --for=condition=Ready --timeout=120s

echo -e "${GREEN}Test workloads deployed successfully!${NC}"
echo -e "${GREEN}Tip: Use 'make test-workloads-cleanup' to remove test workloads${NC}"
