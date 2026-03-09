#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-dev"
POSTGRES_RELEASE="mochi-postgres"
PROMETHEUS_RELEASE="mochi-prometheus"
REDIS_RELEASE="mochi-redis"

echo -e "${GREEN}Setting up Mochi development environment in minikube${NC}\n"

# Check if minikube is running
if ! minikube status &>/dev/null; then
    echo -e "${YELLOW}Minikube is not running. Starting minikube...${NC}"
    minikube start
fi

# Check if Helm is installed
if ! command -v helm &> /dev/null; then
    echo -e "${RED}Helm is not installed. Please install Helm first.${NC}"
    echo "Visit: https://helm.sh/docs/intro/install/"
    exit 1
fi

# Create namespace if it doesn't exist
echo -e "${GREEN}Creating namespace: ${NAMESPACE}${NC}"
kubectl create namespace ${NAMESPACE} >/dev/null 2>&1 || true

# Add Helm repositories
echo -e "${GREEN}Adding Helm repositories...${NC}"
helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
helm repo update >/dev/null 2>&1 || true

# Install PostgreSQL
echo -e "\n${GREEN}Installing PostgreSQL...${NC}"
helm upgrade --install ${POSTGRES_RELEASE} bitnami/postgresql \
    --namespace ${NAMESPACE} \
    --set fullnameOverride=${POSTGRES_RELEASE} \
    --set auth.postgresPassword=mochi \
    --set auth.username=mochi \
    --set auth.password=mochi \
    --set auth.database=mochi \
    --set primary.persistence.size=2Gi \
    --set primary.resources.requests.memory=256Mi \
    --set primary.resources.requests.cpu=250m \
    --wait >/dev/null 2>&1

# Install Prometheus
echo -e "\n${GREEN}Installing Prometheus...${NC}"
helm upgrade --install ${PROMETHEUS_RELEASE} prometheus-community/kube-prometheus-stack \
    --namespace ${NAMESPACE} \
    --set fullnameOverride=${PROMETHEUS_RELEASE} \
    --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=10Gi \
    --set prometheus.prometheusSpec.retention=7d \
    --set prometheus.prometheusSpec.resources.requests.memory=512Mi \
    --set prometheus.prometheusSpec.resources.requests.cpu=500m \
    --set grafana.enabled=false \
    --set alertmanager.enabled=false \
    --wait >/dev/null 2>&1

# Install Redis
echo -e "\n${GREEN}Installing Redis...${NC}"
helm upgrade --install ${REDIS_RELEASE} bitnami/redis \
    --namespace ${NAMESPACE} \
    --set fullnameOverride=${REDIS_RELEASE} \
    --set auth.enabled=true \
    --set auth.password=mochi \
    --set master.persistence.size=2Gi \
    --set master.resources.requests.memory=256Mi \
    --set master.resources.requests.cpu=250m \
    --set replica.replicaCount=0 \
    --wait >/dev/null 2>&1

# Print Services info
echo -e "\n${GREEN}Installation complete!${NC}\n"
echo -e "${YELLOW}Services Information:${NC}\n"

# PostgreSQL
POSTGRES_SVC="${POSTGRES_RELEASE}"
echo -e "${GREEN}PostgreSQL:${NC}"
echo "  Host: ${POSTGRES_SVC}.${NAMESPACE}.svc.cluster.local"
echo "  Port: 5432"
echo "  Database: mochi"
echo "  Username: mochi"
echo "  Password: mochi"
echo ""

# Prometheus
PROMETHEUS_SVC="${PROMETHEUS_RELEASE}-prometheus"
echo -e "\n${GREEN}Prometheus:${NC}"
echo "  URL: http://${PROMETHEUS_SVC}.${NAMESPACE}.svc.cluster.local:9090"
echo ""

# Redis
REDIS_SVC="${REDIS_RELEASE}-master"
echo -e "\n${GREEN}Redis:${NC}"
echo "  Host: ${REDIS_SVC}.${NAMESPACE}.svc.cluster.local"
echo "  Port: 6379"
echo "  Password: mochi"
echo "  Database: 0"
echo ""

echo -e "\n${GREEN}Tip: Use 'make dev-env-status' to check the status of services${NC}"
echo -e "${GREEN}Tip: Use 'make dev-env-clean' to remove all services${NC}"
