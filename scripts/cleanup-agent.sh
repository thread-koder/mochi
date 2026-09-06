#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/agent/deploy"

echo -e "${YELLOW}Cleaning up mochi-agent${NC}\n"

# Confirm deletion
read -p "Are you sure you want to remove mochi-agent? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}Aborted.${NC}"
    exit 0
fi

# Check if namespace exists
if ! kubectl get namespace "${NAMESPACE}" &>/dev/null; then
    echo -e "${YELLOW}Namespace '${NAMESPACE}' does not exist. Nothing to clean up.${NC}"
    exit 0
fi

echo -e "${GREEN}Removing mochi-agent...${NC}"
kubectl delete -f "${DEPLOY_DIR}/podmonitor.yaml" --ignore-not-found=true >/dev/null 2>&1 || true
kubectl delete -f "${DEPLOY_DIR}/daemonset.yaml" --ignore-not-found=true >/dev/null 2>&1 || true
kubectl delete -f "${DEPLOY_DIR}/rbac.yaml" --ignore-not-found=true >/dev/null 2>&1 || true

echo -e "\n${GREEN}Cleanup complete!${NC}"
