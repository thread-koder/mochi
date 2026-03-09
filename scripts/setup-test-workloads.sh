#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKLOADS_FILE="${SCRIPT_DIR}/test-workloads.yaml"

echo -e "${GREEN}Setting up test workloads in minikube${NC}\n"

# Check if minikube is running
if ! minikube status &>/dev/null; then
    echo -e "${RED}Minikube is not running. Please start minikube first.${NC}"
    exit 1
fi

# Check if namespace exists
if ! kubectl get namespace ${NAMESPACE} &>/dev/null; then
    echo -e "${YELLOW}Namespace '${NAMESPACE}' does not exist. Creating it...${NC}"
    kubectl create namespace ${NAMESPACE} >/dev/null 2>&1
fi

# Check if workloads file exists
if [ ! -f "${WORKLOADS_FILE}" ]; then
    echo -e "${RED}Workloads file not found: ${WORKLOADS_FILE}${NC}"
    exit 1
fi

# Deploy test workloads
echo -e "${GREEN}Deploying test workloads...${NC}"
kubectl apply -f "${WORKLOADS_FILE}" --namespace ${NAMESPACE} >/dev/null 2>&1

echo -e "${GREEN}Waiting for workloads to be ready...${NC}"
kubectl wait --for=condition=available --timeout=60s deployment/test-deployment -n ${NAMESPACE} >/dev/null 2>&1 || true
kubectl wait --for=condition=ready --timeout=60s pod/test-standalone-pod -n ${NAMESPACE} >/dev/null 2>&1 || true

echo -e "${GREEN}Test workloads deployed successfully!${NC}"
echo -e "${GREEN}Tip: Use 'make test-workloads-clean' to remove test workloads${NC}"