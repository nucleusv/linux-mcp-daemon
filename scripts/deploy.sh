#!/bin/bash
# scripts/deploy.sh
set -e

# Change to the root directory of the project
cd "$(dirname "$0")/.."

# Generate a unique tag to prevent Kubernetes from aggressively caching the local image
TAG=$(date +%s)

echo "Building Docker image 'linux-mcp-daemon:$TAG'..."
docker build -t linux-mcp-daemon:$TAG .

echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

echo "Setting new image tag and force restarting deployment..."
kubectl set image deployment/linux-mcp-daemon daemon=linux-mcp-daemon:$TAG

echo "Waiting for deployment to be ready..."
kubectl rollout status deployment/linux-mcp-daemon

echo "Deployment complete. The service should be accessible at http://localhost:9090"
