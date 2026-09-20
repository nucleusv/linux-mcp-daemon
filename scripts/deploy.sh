#!/bin/bash
# scripts/deploy.sh
set -e

# Change to the root directory of the project
cd "$(dirname "$0")/.."

echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

echo "Waiting for deployment to be ready..."
kubectl rollout status deployment/linux-mcp-daemon

echo "Deployment complete. The service should be accessible at http://localhost:9090"
