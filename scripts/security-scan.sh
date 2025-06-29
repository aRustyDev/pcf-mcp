#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running Security Scans...${NC}"

# Track if any scan fails
SCAN_FAILED=0

# 1. Check for gosec (Go security checker)
echo -e "\n${GREEN}Running gosec...${NC}"
if command -v gosec &> /dev/null; then
    if gosec -fmt json -out gosec-report.json ./... 2>/dev/null; then
        echo -e "${GREEN}✓ gosec scan passed${NC}"
    else
        echo -e "${RED}✗ gosec found security issues${NC}"
        cat gosec-report.json | jq '.Issues[] | {severity: .severity, confidence: .confidence, details: .details, file: .file, line: .line}'
        SCAN_FAILED=1
    fi
else
    echo -e "${YELLOW}gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest${NC}"
fi

# 2. Check for vulnerable dependencies
echo -e "\n${GREEN}Checking for vulnerable dependencies...${NC}"
if command -v nancy &> /dev/null; then
    if go list -json -m all | nancy sleuth; then
        echo -e "${GREEN}✓ No vulnerable dependencies found${NC}"
    else
        echo -e "${RED}✗ Vulnerable dependencies detected${NC}"
        SCAN_FAILED=1
    fi
else
    echo -e "${YELLOW}nancy not installed. Install with: go install github.com/sonatype-nexus-community/nancy@latest${NC}"
fi

# 3. Run go mod audit
echo -e "\n${GREEN}Running go mod audit...${NC}"
if go list -m -u -json all | grep -q '"Update"'; then
    echo -e "${YELLOW}⚠ Updates available for dependencies:${NC}"
    go list -m -u all | grep -v "^github.com/analyst/pcf-mcp"
fi

# 4. Check for secrets in code
echo -e "\n${GREEN}Checking for secrets in code...${NC}"
if command -v trufflehog &> /dev/null; then
    if trufflehog filesystem . --no-update --json --only-verified 2>/dev/null | jq -s 'length == 0' > /dev/null; then
        echo -e "${GREEN}✓ No secrets found in code${NC}"
    else
        echo -e "${RED}✗ Potential secrets found in code${NC}"
        trufflehog filesystem . --no-update --json --only-verified 2>/dev/null | jq '.SourceMetadata'
        SCAN_FAILED=1
    fi
else
    # Fall back to basic pattern matching
    echo -e "${YELLOW}trufflehog not installed. Using basic pattern matching...${NC}"
    
    # Common patterns for secrets
    if grep -rEn "(api_key|apikey|api-key|secret|password|passwd|pwd|token|bearer|private_key|ssh_key)" . \
        --exclude-dir=.git \
        --exclude-dir=vendor \
        --exclude-dir=node_modules \
        --exclude="*.json" \
        --exclude="*.md" \
        --exclude="security-scan.sh" | \
        grep -vE "(test|example|sample|fake|dummy|placeholder|config\.go|_test\.go)" | \
        grep -iE "(=|:)\s*['\"][^'\"]{8,}['\"]"; then
        echo -e "${RED}✗ Potential secrets found in code${NC}"
        SCAN_FAILED=1
    else
        echo -e "${GREEN}✓ No obvious secrets found in code${NC}"
    fi
fi

# 5. Check Docker image security
echo -e "\n${GREEN}Checking Docker image security...${NC}"
if command -v trivy &> /dev/null && [ -f Dockerfile ]; then
    if docker images | grep -q "pcf-mcp"; then
        if trivy image --severity HIGH,CRITICAL pcf-mcp:latest; then
            echo -e "${GREEN}✓ Docker image scan passed${NC}"
        else
            echo -e "${RED}✗ Docker image has vulnerabilities${NC}"
            SCAN_FAILED=1
        fi
    else
        echo -e "${YELLOW}Docker image not built. Build with: docker build -t pcf-mcp:latest .${NC}"
    fi
else
    echo -e "${YELLOW}trivy not installed or Dockerfile not found${NC}"
fi

# 6. SAST with semgrep
echo -e "\n${GREEN}Running SAST with semgrep...${NC}"
if command -v semgrep &> /dev/null; then
    if semgrep --config=auto --json -o semgrep-report.json . 2>/dev/null && \
       [ $(jq '.results | length' semgrep-report.json) -eq 0 ]; then
        echo -e "${GREEN}✓ semgrep SAST scan passed${NC}"
    else
        echo -e "${RED}✗ semgrep found issues${NC}"
        jq '.results[] | {path: .path, message: .extra.message, severity: .extra.severity}' semgrep-report.json 2>/dev/null || true
        SCAN_FAILED=1
    fi
else
    echo -e "${YELLOW}semgrep not installed. Install with: pip install semgrep${NC}"
fi

# 7. License compliance check
echo -e "\n${GREEN}Checking license compliance...${NC}"
if command -v go-licenses &> /dev/null; then
    if go-licenses check ./... --disallowed_types=forbidden,unknown 2>/dev/null; then
        echo -e "${GREEN}✓ License compliance check passed${NC}"
    else
        echo -e "${YELLOW}⚠ License compliance issues found${NC}"
        go-licenses report ./... 2>/dev/null || true
    fi
else
    echo -e "${YELLOW}go-licenses not installed. Install with: go install github.com/google/go-licenses@latest${NC}"
fi

# Summary
echo -e "\n${YELLOW}Security Scan Summary:${NC}"
if [ $SCAN_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All security scans passed${NC}"
    exit 0
else
    echo -e "${RED}✗ Security issues found. Please review and fix before deployment.${NC}"
    exit 1
fi