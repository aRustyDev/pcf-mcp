#!/bin/bash
# Test Docker build and push locally
# This simulates what the GitHub Action does

set -euo pipefail

echo "🧪 Testing Docker workflow locally"
echo "================================="
echo ""

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker Desktop."
    exit 1
fi

if ! command -v op &> /dev/null; then
    echo "❌ 1Password CLI not found. Please install it:"
    echo "   brew install --cask 1password-cli"
    exit 1
fi

# Load credentials from 1Password
echo "🔐 Loading credentials from 1Password..."
export DOCKERHUB_USERNAME=$(op read "op://CI-CD/DockerHub/username")
export DOCKERHUB_TOKEN=$(op read "op://CI-CD/DockerHub/token")
export GHCR_TOKEN=$(op read "op://CI-CD/GitHub/packages_write_token")

if [ -z "$DOCKERHUB_USERNAME" ] || [ -z "$DOCKERHUB_TOKEN" ]; then
    echo "❌ Failed to load Docker Hub credentials from 1Password"
    exit 1
fi

echo "✓ Credentials loaded successfully"

# Docker Hub login
echo ""
echo "🐳 Logging in to Docker Hub..."
echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin

# GHCR login
echo ""
echo "🐙 Logging in to GitHub Container Registry..."
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$USER" --password-stdin

# Build image
echo ""
echo "🔨 Building Docker image..."
docker build -t pcf-mcp:test .

# Tag images
echo ""
echo "🏷️  Tagging images..."
docker tag pcf-mcp:test "$DOCKERHUB_USERNAME/pcf-mcp:test-local"
docker tag pcf-mcp:test "ghcr.io/$USER/pcf-mcp:test-local"

# Optional: Push images
echo ""
echo "📤 Ready to push images:"
echo "   docker push $DOCKERHUB_USERNAME/pcf-mcp:test-local"
echo "   docker push ghcr.io/$USER/pcf-mcp:test-local"
echo ""
read -p "Push images? (y/N) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Pushing to Docker Hub..."
    docker push "$DOCKERHUB_USERNAME/pcf-mcp:test-local"
    
    echo "Pushing to GHCR..."
    docker push "ghcr.io/$USER/pcf-mcp:test-local"
    
    echo ""
    echo "✅ Images pushed successfully!"
    echo ""
    echo "Pull commands:"
    echo "   docker pull $DOCKERHUB_USERNAME/pcf-mcp:test-local"
    echo "   docker pull ghcr.io/$USER/pcf-mcp:test-local"
else
    echo "Skipping push."
fi

# Cleanup
echo ""
echo "🧹 Cleaning up..."
docker rmi "pcf-mcp:test" "$DOCKERHUB_USERNAME/pcf-mcp:test-local" "ghcr.io/$USER/pcf-mcp:test-local" 2>/dev/null || true

echo ""
echo "✅ Test completed!"