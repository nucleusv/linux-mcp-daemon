#!/bin/bash
# scripts/deploy.sh
set -e

# Change to the root directory of the project
cd "$(dirname "$0")/.."

NAMESPACE="linux-mcp-daemon-by-claude"

# Generate a unique tag to prevent Kubernetes from aggressively caching the local image
TAG=$(date +%s)

IMAGE="linux-mcp-daemon-by-claude"

echo "Building Docker image '$IMAGE:$TAG'..."
# Also tag :local, the placeholder image name committed in k8s/deployment.yaml.
# Without this, a plain `kubectl apply -f k8s/deployment.yaml` (bypassing the
# `kubectl set image` step below) would reset the running pod to whatever
# ancient build :local last pointed to - which crash-looped us once already.
docker build -t $IMAGE:$TAG -t $IMAGE:local .

# Keep the local linuxctl binary in sync with every daemon image build.
./scripts/build-cli.sh

echo "Ensuring namespace '$NAMESPACE' exists..."
kubectl apply -f k8s/namespace.yaml

echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

echo "Setting new image tag and force restarting deployment..."
kubectl set image deployment/linux-mcp-daemon daemon=$IMAGE:$TAG -n "$NAMESPACE"

echo "Waiting for deployment to be ready..."
kubectl rollout status deployment/linux-mcp-daemon -n "$NAMESPACE"

echo "Deployment complete. The service should be accessible at http://localhost:9091"
