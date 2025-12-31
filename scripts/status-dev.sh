#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="mochi-dev"
POSTGRES_RELEASE="mochi-postgres"
PROMETHEUS_RELEASE="mochi-prometheus"

echo -e "${BLUE}📊 Mochi Development Environment Status${NC}\n"

# Check if namespace exists
if ! kubectl get namespace ${NAMESPACE} &>/dev/null; then
    echo -e "${RED}❌ Namespace '${NAMESPACE}' does not exist${NC}"
    echo -e "${YELLOW}💡 Run 'make dev-env-setup' to create the environment${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Namespace '${NAMESPACE}' exists${NC}\n"

# Check Helm releases
echo -e "${BLUE}📦 Helm Releases:${NC}"
helm list -n ${NAMESPACE}

echo ""

# Check PostgreSQL
echo -e "${BLUE}🐘 PostgreSQL:${NC}"
if helm list -n ${NAMESPACE} | grep -q ${POSTGRES_RELEASE}; then
    POSTGRES_POD=$(kubectl get pods -n ${NAMESPACE} -l app.kubernetes.io/name=postgresql -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$POSTGRES_POD" ]; then
        STATUS=$(kubectl get pod ${POSTGRES_POD} -n ${NAMESPACE} -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$STATUS" = "Running" ]; then
            echo -e "  Status: ${GREEN}Running${NC}"
        else
            echo -e "  Status: ${YELLOW}${STATUS}${NC}"
        fi
        POSTGRES_SVC="${POSTGRES_RELEASE}"
        echo -e "  Service: ${POSTGRES_SVC}.${NAMESPACE}.svc.cluster.local:5432"
    else
        echo -e "  Status: ${YELLOW}Pod not found${NC}"
    fi
else
    echo -e "  Status: ${RED}Not installed${NC}"
fi

echo ""

# Check Prometheus
echo -e "${BLUE}📊 Prometheus:${NC}"
if helm list -n ${NAMESPACE} | grep -q ${PROMETHEUS_RELEASE}; then
    PROMETHEUS_POD=$(kubectl get pods -n ${NAMESPACE} -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$PROMETHEUS_POD" ]; then
        STATUS=$(kubectl get pod ${PROMETHEUS_POD} -n ${NAMESPACE} -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$STATUS" = "Running" ]; then
            echo -e "  Status: ${GREEN}Running${NC}"
        else
            echo -e "  Status: ${YELLOW}${STATUS}${NC}"
        fi
        PROMETHEUS_SVC="${PROMETHEUS_RELEASE}-prometheus"
        echo -e "  Service: http://${PROMETHEUS_SVC}.${NAMESPACE}.svc.cluster.local:9090"
    else
        echo -e "  Status: ${YELLOW}Pod not found${NC}"
    fi
else
    echo -e "  Status: ${RED}Not installed${NC}"
fi

echo ""

# Show pods
echo -e "${BLUE}🔍 Pods in namespace:${NC}"
kubectl get pods -n ${NAMESPACE}

echo ""

# Show services
echo -e "\n${BLUE}🌐 Services in namespace:${NC}"
kubectl get svc -n ${NAMESPACE}
