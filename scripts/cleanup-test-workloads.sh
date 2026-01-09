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

echo -e "${YELLOW}🧹 Cleaning up test workloads${NC}\n"

# Confirm deletion
read -p "Are you sure you want to remove all test workloads? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}Aborted.${NC}"
    exit 0
fi

# Check if namespace exists
if ! kubectl get namespace ${NAMESPACE} &>/dev/null; then
    echo -e "${YELLOW}⚠️  Namespace '${NAMESPACE}' does not exist. Nothing to clean up.${NC}"
    exit 0
fi

# Remove workloads using the YAML file if it exists
if [ -f "${WORKLOADS_FILE}" ]; then
    echo -e "${GREEN}🗑️  Removing test workloads...${NC}"
    kubectl delete -f "${WORKLOADS_FILE}" --namespace ${NAMESPACE} --ignore-not-found=true >/dev/null 2>&1 || true
else
    # Fallback: delete by name if YAML file doesn't exist
    echo -e "${YELLOW}⚠️  Workloads file not found, deleting by name...${NC}"
    kubectl delete deployment test-deployment -n ${NAMESPACE} --ignore-not-found=true >/dev/null 2>&1 || true
    kubectl delete daemonset test-daemonset -n ${NAMESPACE} --ignore-not-found=true >/dev/null 2>&1 || true
    kubectl delete pod test-standalone-pod -n ${NAMESPACE} --ignore-not-found=true >/dev/null 2>&1 || true
fi

echo -e "\n${GREEN}✅ Cleanup complete!${NC}"