# 1Password CI/CD Setup Guide

This guide explains how to configure 1Password for the Docker image publishing workflow.

## Prerequisites

1. 1Password account with appropriate permissions
2. 1Password CLI installed locally (for initial setup)
3. GitHub repository admin access

## Setup Steps

### 1. Create 1Password Service Account

1. Log in to your 1Password account
2. Go to **Integrations** → **Service Accounts**
3. Click **Create Service Account**
4. Name it: `GitHub Actions - PCF-MCP`
5. Save the generated token securely

### 2. Create 1Password Vaults and Items

Create a vault named `CI-CD` (or use an existing one) with the following items:

#### DockerHub Item
- **Name**: `DockerHub`
- **Fields**:
  - `username`: Your Docker Hub username
  - `token`: Docker Hub access token (create at https://hub.docker.com/settings/security)

#### GitHub Item
- **Name**: `GitHub`
- **Fields**:
  - `packages_write_token`: GitHub PAT with `write:packages` scope

### 3. Grant Service Account Access

1. Go to the `CI-CD` vault settings
2. Add the service account with **Read** permissions
3. Ensure the service account can access both items

### 4. Configure GitHub Repository Secrets

Add the following secret to your GitHub repository:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**
3. Add:
   - **Name**: `OP_SERVICE_ACCOUNT_TOKEN`
   - **Value**: The service account token from step 1

## Vault Structure

The workflow expects the following 1Password structure:

```
CI-CD (vault)
├── DockerHub (item)
│   ├── username (field)
│   └── token (field)
└── GitHub (item)
    └── packages_write_token (field)
```

## Creating Required Tokens

### Docker Hub Access Token

1. Log in to [Docker Hub](https://hub.docker.com)
2. Go to **Account Settings** → **Security**
3. Click **New Access Token**
4. Description: `GitHub Actions - PCF-MCP`
5. Permissions: **Read, Write, Delete**
6. Copy the token and save it in 1Password

### GitHub Personal Access Token

1. Go to GitHub **Settings** → **Developer settings** → **Personal access tokens** → **Fine-grained tokens**
2. Click **Generate new token**
3. Name: `PCF-MCP Docker Registry`
4. Expiration: Set as needed (recommend 90 days with rotation)
5. Repository access: Select the `pcf-mcp` repository
6. Permissions:
   - **Packages**: Write
7. Generate and save the token in 1Password

## Testing the Setup

To test your 1Password configuration locally:

```bash
# Install 1Password CLI
brew install --cask 1password-cli

# Sign in
eval $(op signin)

# Test retrieving secrets
op read "op://CI-CD/DockerHub/username"
op read "op://CI-CD/DockerHub/token"
op read "op://CI-CD/GitHub/packages_write_token"
```

## Security Best Practices

1. **Rotate tokens regularly** - Set calendar reminders for token rotation
2. **Use least privilege** - Only grant necessary permissions
3. **Audit access** - Regularly review who has access to the vault
4. **Enable 2FA** - Ensure all accounts have 2FA enabled
5. **Monitor usage** - Check 1Password audit logs for unusual activity

## Troubleshooting

### Common Issues

1. **"Could not find item"** - Check vault and item names match exactly
2. **"Permission denied"** - Verify service account has vault access
3. **"Invalid token"** - Token may be expired or incorrectly copied
4. **"Login failed"** - Docker Hub may require 2FA token; use access token instead

### Debug Commands

```bash
# List accessible vaults
op vault list

# List items in vault
op item list --vault="CI-CD"

# Get item details
op item get "DockerHub" --vault="CI-CD"
```

## Workflow Behavior

The workflow will:
1. Trigger on pushes to `main` or merged PRs
2. Load credentials from 1Password
3. Build multi-platform images (amd64, arm64)
4. Push to both Docker Hub and GitHub Container Registry
5. Sign images with Cosign
6. Generate SBOM (Software Bill of Materials)
7. Run vulnerability scans with Trivy and Grype
8. Upload results to GitHub Security tab

## Image Tagging Strategy

Images are tagged with:
- `latest` - Latest build from main branch
- `main-<sha>` - Git commit SHA
- `YYYYMMDD-HHmmss` - Timestamp
- `v1.2.3`, `v1.2`, `v1` - Semantic versions (when tagged)