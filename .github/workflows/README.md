# GitHub Actions CI/CD Workflows

This directory contains the CI/CD workflows for the PCF-MCP project.

## Workflows

### 1. Continuous Integration (`ci.yml`)
**Triggers**: Push to main/develop, Pull requests to main

- **Test**: Runs unit tests on Go 1.22 and 1.23
- **Lint**: Runs golangci-lint, gofmt, and go vet
- **Security**: Runs Gosec and Nancy vulnerability scans
- **Build**: Builds the binary for linux/amd64
- **Integration Tests**: Runs integration test suite
- **Benchmarks**: Runs performance benchmarks (main branch only)

### 2. Docker Publish (`docker-publish.yml`)
**Triggers**: Push to main, Merged pull requests to main

- Builds multi-platform Docker images (amd64, arm64)
- Pushes to Docker Hub and GitHub Container Registry
- Signs images with Cosign
- Generates SBOM (Software Bill of Materials)
- Runs vulnerability scans with Trivy and Grype
- Uses 1Password for credential management

### 3. Release (`release.yml`)
**Triggers**: Version tags (v*.*.*)

- Creates GitHub releases with GoReleaser
- Builds binaries for multiple platforms
- Creates and signs checksums
- Builds and publishes versioned Docker images
- Generates release notes from commits

## Setup Requirements

### 1. GitHub Repository Secrets

Add the following secret to your repository:
- `OP_SERVICE_ACCOUNT_TOKEN` - 1Password service account token

### 2. 1Password Configuration

See [1password-setup.md](./1password-setup.md) for detailed setup instructions.

Quick setup:
```bash
# Run the setup helper script
./.github/workflows/scripts/setup-1password.sh
```

### 3. Required Tokens

#### Docker Hub Access Token
1. Go to [Docker Hub Security Settings](https://hub.docker.com/settings/security)
2. Create a new access token with Read, Write, Delete permissions
3. Store in 1Password as `op://CI-CD/DockerHub/token`

#### GitHub Personal Access Token
1. Go to [GitHub Tokens](https://github.com/settings/tokens)
2. Create a fine-grained PAT with `write:packages` permission
3. Store in 1Password as `op://CI-CD/GitHub/packages_write_token`

## Workflow Features

### Security
- All credentials stored in 1Password
- Container image signing with Cosign
- Vulnerability scanning with multiple tools
- SARIF upload to GitHub Security tab
- Dependency scanning with Nancy

### Quality
- Multi-version Go testing
- Comprehensive linting
- Integration test suite
- Benchmark tracking
- Code coverage with Codecov

### Automation
- Automatic version tagging from Git tags
- Multi-platform builds (linux, darwin, windows)
- Multi-architecture Docker images
- Automatic SBOM generation
- Release note generation from commits

## Docker Image Tags

Images are published with the following tags:
- `latest` - Latest build from main branch
- `main-<short-sha>` - Git commit SHA
- `YYYYMMDD-HHmmss` - Build timestamp
- `v1.2.3` - Semantic version (releases only)
- `v1.2` - Minor version (releases only)
- `v1` - Major version (releases only)

## Verifying Published Images

### Pull Images
```bash
# Docker Hub
docker pull arustydev/pcf-mcp:latest

# GitHub Container Registry
docker pull ghcr.io/arustydev/pcf-mcp:latest
```

### Verify Signatures
```bash
# Verify Docker Hub image
cosign verify arustydev/pcf-mcp:latest

# Verify GHCR image
cosign verify ghcr.io/arustydev/pcf-mcp:latest
```

### Inspect SBOM
```bash
# Download SBOM
cosign download sbom arustydev/pcf-mcp:latest

# View SBOM
cosign download sbom arustydev/pcf-mcp:latest | jq
```

## Monitoring Workflows

### Workflow Status
- Check [Actions tab](../../actions) for workflow runs
- Each workflow provides detailed logs
- Failed workflows send notifications (if configured)

### Security Alerts
- Check [Security tab](../../security) for vulnerability reports
- Trivy and Grype results appear under "Code scanning"
- Dependency vulnerabilities tracked automatically

## Troubleshooting

### Common Issues

1. **1Password authentication fails**
   - Verify `OP_SERVICE_ACCOUNT_TOKEN` is set correctly
   - Check service account has access to CI-CD vault
   - Ensure item paths match exactly (case-sensitive)

2. **Docker push fails**
   - Verify Docker Hub token has correct permissions
   - Check Docker Hub username is correct
   - Ensure repository exists on Docker Hub

3. **GHCR push fails**
   - Verify GitHub PAT has `write:packages` scope
   - Check PAT hasn't expired
   - Ensure PAT has access to the repository

4. **Cosign signing fails**
   - Cosign uses keyless signing (no keys needed)
   - Requires `id-token: write` permission
   - Only works in GitHub Actions environment

### Debug Commands

Test 1Password access locally:
```bash
# Check vault access
op vault list

# Test reading secrets
op read "op://CI-CD/DockerHub/username"
op read "op://CI-CD/DockerHub/token"
```

Test Docker login:
```bash
# Docker Hub
echo $DOCKERHUB_TOKEN | docker login -u $DOCKERHUB_USERNAME --password-stdin

# GHCR
echo $GHCR_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin
```

## Best Practices

1. **Regular Token Rotation**
   - Rotate Docker Hub tokens every 90 days
   - Rotate GitHub PATs every 90 days
   - Update in 1Password, no workflow changes needed

2. **Monitor Usage**
   - Check Docker Hub rate limits
   - Monitor GitHub Actions minutes
   - Review security scan results regularly

3. **Version Management**
   - Use semantic versioning for releases
   - Tag releases trigger special workflows
   - Keep CHANGELOG.md updated

4. **Security**
   - Never commit credentials
   - Use 1Password for all secrets
   - Enable 2FA on all accounts
   - Review workflow permissions regularly