#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-system"
POSTGRES_RELEASE="mochi-postgres"
PROMETHEUS_RELEASE="mochi-prometheus"
REDIS_RELEASE="mochi-redis"

echo -e "${YELLOW}Cleaning up development environment dependencies${NC}\n"

# Confirm deletion
read -p "Are you sure you want to remove all development environment dependencies? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}Aborted.${NC}"
    exit 0
fi

# Check if namespace exists
if ! kubectl get namespace ${NAMESPACE} &>/dev/null; then
    echo -e "${YELLOW}Namespace '${NAMESPACE}' does not exist. Nothing to clean up.${NC}"
    exit 0
fi

# Uninstall Helm releases
echo -e "${GREEN}Uninstalling Helm releases...${NC}"

if helm list -n ${NAMESPACE} | grep -q ${POSTGRES_RELEASE}; then
    echo -e "${YELLOW}Removing PostgreSQL...${NC}"
    helm uninstall ${POSTGRES_RELEASE} -n ${NAMESPACE} || true
else
    echo -e "${YELLOW}PostgreSQL not found, skipping...${NC}"
fi

if helm list -n ${NAMESPACE} | grep -q ${PROMETHEUS_RELEASE}; then
    echo -e "${YELLOW}Removing Prometheus...${NC}"
    helm uninstall ${PROMETHEUS_RELEASE} -n ${NAMESPACE} || true
else
    echo -e "${YELLOW}Prometheus not found, skipping...${NC}"
fi

if helm list -n ${NAMESPACE} | grep -q ${REDIS_RELEASE}; then
    echo -e "${YELLOW}Removing Redis...${NC}"
    helm uninstall ${REDIS_RELEASE} -n ${NAMESPACE} || true
else
    echo -e "${YELLOW}Redis not found, skipping...${NC}"
fi

# Optionally remove namespace
read -p "Do you want to remove the namespace '${NAMESPACE}'? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Removing namespace...${NC}"
    kubectl delete namespace ${NAMESPACE} || true
    echo -e "${GREEN}Namespace removed${NC}"
fi

echo -e "\n${GREEN}Cleanup complete!${NC}"
