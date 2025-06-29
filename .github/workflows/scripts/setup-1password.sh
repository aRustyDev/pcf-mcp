#!/bin/bash
# Script to help set up 1Password items for CI/CD
# This script should be run locally, not in CI

set -euo pipefail

echo "🔐 PCF-MCP 1Password CI/CD Setup Helper"
echo "======================================="
echo ""
echo "This script will help you create the required 1Password items."
echo "Make sure you have the 1Password CLI installed and are signed in."
echo ""

# Check if op CLI is installed
if ! command -v op &> /dev/null; then
    echo "❌ 1Password CLI not found. Please install it first:"
    echo "   brew install --cask 1password-cli"
    exit 1
fi

# Check if user is signed in
if ! op account list &> /dev/null; then
    echo "❌ Not signed in to 1Password. Please run:"
    echo "   op signin"
    exit 1
fi

# Function to create or update an item
create_or_update_item() {
    local vault=$1
    local item_name=$2
    local template=$3

    echo "📝 Checking for item '$item_name' in vault '$vault'..."

    if op item get "$item_name" --vault="$vault" &> /dev/null; then
        echo "   ✓ Item already exists"
    else
        echo "   → Creating new item..."
        echo "$template" | op item create --vault="$vault" -
        echo "   ✓ Item created"
    fi
}

# Create CI-CD vault if it doesn't exist
echo "🗂️  Checking for CI-CD vault..."
if ! op vault get "CI-CD" &> /dev/null; then
    echo "   → Creating CI-CD vault..."
    op vault create "CI-CD"
    echo "   ✓ Vault created"
else
    echo "   ✓ Vault already exists"
fi

# DockerHub item template
DOCKERHUB_TEMPLATE='{
  "title": "DockerHub",
  "category": "API Credential",
  "fields": [
    {
      "id": "username",
      "type": "STRING",
      "label": "username",
      "value": "REPLACE_WITH_YOUR_DOCKERHUB_USERNAME"
    },
    {
      "id": "token",
      "type": "CONCEALED",
      "label": "token",
      "value": "REPLACE_WITH_YOUR_DOCKERHUB_TOKEN"
    }
  ]
}'

# GitHub item template
GITHUB_TEMPLATE='{
  "title": "GitHub",
  "category": "API Credential",
  "fields": [
    {
      "id": "packages_write_token",
      "type": "CONCEALED",
      "label": "packages_write_token",
      "value": "REPLACE_WITH_YOUR_GITHUB_PAT"
    }
  ]
}'

# GoReleaser item template
GORELEASER_TEMPLATE='{
  "title": "GoReleaser",
  "category": "API Credential",
  "fields": [
    {
      "id": "signing_key",
      "type": "CONCEALED",
      "label": "signing_key",
      "value": "REPLACE_WITH_YOUR_SIGNING_KEY"
    }
  ]
}'

# Create items
create_or_update_item "CI-CD" "DockerHub" "$DOCKERHUB_TEMPLATE"
create_or_update_item "CI-CD" "GitHub" "$GITHUB_TEMPLATE"
create_or_update_item "CI-CD" "GoReleaser" "$GORELEASER_TEMPLATE"

echo ""
echo "✅ 1Password structure created!"
echo ""
echo "⚠️  IMPORTANT: You need to update the placeholder values:"
echo ""
echo "1. DockerHub:"
echo "   - Go to https://hub.docker.com/settings/security"
echo "   - Create a new access token"
echo "   - Update the 'username' and 'token' fields in 1Password"
echo ""
echo "2. GitHub:"
echo "   - Go to https://github.com/settings/tokens"
echo "   - Create a fine-grained PAT with 'write:packages' permission"
echo "   - Update the 'packages_write_token' field in 1Password"
echo ""
echo "3. GoReleaser (optional):"
echo "   - Generate a signing key if you want to sign releases"
echo "   - Update the 'signing_key' field in 1Password"
echo ""
echo "To update a field:"
echo "   op item edit DockerHub username=<your-username> --vault=CI-CD"
echo "   op item edit DockerHub token=<your-token> --vault=CI-CD"
echo ""
echo "To verify your setup:"
echo "   op read \"op://PCF-MCP/DockerHub/username\""
echo "   op read \"op://PCF-MCP/DockerHub/token\""
echo "   op read \"op://PCF-MCP/GitHub/packages_write_token\""
