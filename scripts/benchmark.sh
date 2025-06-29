#!/bin/bash
set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create benchmark results directory
RESULTS_DIR="benchmark-results"
mkdir -p "$RESULTS_DIR"

# Timestamp for results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/bench_${TIMESTAMP}.txt"
COMPARISON_FILE="$RESULTS_DIR/comparison_${TIMESTAMP}.txt"

echo -e "${YELLOW}Running Performance Benchmarks...${NC}"

# Run benchmarks and save results
echo -e "\n${GREEN}Running all benchmarks...${NC}"
go test -bench=. -benchmem -benchtime=10s -cpu=1,2,4 ./internal/mcp ./internal/pcf | tee "$RESULTS_FILE"

# Run specific benchmarks with more detail
echo -e "\n${GREEN}Running detailed MCP server benchmarks...${NC}"
go test -bench=. -benchmem -benchtime=30s -cpuprofile=cpu.prof -memprofile=mem.prof \
    github.com/analyst/pcf-mcp/internal/mcp

# Generate comparison if previous results exist
LATEST_RESULT=$(ls -t "$RESULTS_DIR"/bench_*.txt 2>/dev/null | grep -v "$RESULTS_FILE" | head -1)
if [ -n "$LATEST_RESULT" ]; then
    echo -e "\n${GREEN}Comparing with previous results...${NC}"
    if command -v benchstat &> /dev/null; then
        benchstat "$LATEST_RESULT" "$RESULTS_FILE" | tee "$COMPARISON_FILE"
    else
        echo -e "${YELLOW}benchstat not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest${NC}"
    fi
fi

# Analyze CPU profile if pprof is available
if [ -f cpu.prof ] && command -v go &> /dev/null; then
    echo -e "\n${GREEN}Analyzing CPU profile...${NC}"
    go tool pprof -top -cum cpu.prof | head -20
    
    # Generate CPU profile graph
    if command -v dot &> /dev/null; then
        go tool pprof -png cpu.prof > "$RESULTS_DIR/cpu_profile_${TIMESTAMP}.png"
        echo -e "${GREEN}CPU profile graph saved to $RESULTS_DIR/cpu_profile_${TIMESTAMP}.png${NC}"
    fi
fi

# Analyze memory profile
if [ -f mem.prof ] && command -v go &> /dev/null; then
    echo -e "\n${GREEN}Analyzing memory profile...${NC}"
    go tool pprof -top -cum mem.prof | head -20
    
    # Generate memory profile graph
    if command -v dot &> /dev/null; then
        go tool pprof -png mem.prof > "$RESULTS_DIR/mem_profile_${TIMESTAMP}.png"
        echo -e "${GREEN}Memory profile graph saved to $RESULTS_DIR/mem_profile_${TIMESTAMP}.png${NC}"
    fi
fi

# Run load test if k6 is available
if command -v k6 &> /dev/null && [ -f scripts/load-test.js ]; then
    echo -e "\n${GREEN}Running load test with k6...${NC}"
    k6 run --out json="$RESULTS_DIR/k6_${TIMESTAMP}.json" scripts/load-test.js
else
    echo -e "${YELLOW}k6 not installed or load-test.js not found. Skipping load test.${NC}"
fi

# Summary
echo -e "\n${BLUE}Benchmark Summary:${NC}"
echo "Results saved to: $RESULTS_FILE"
if [ -n "$COMPARISON_FILE" ] && [ -f "$COMPARISON_FILE" ]; then
    echo "Comparison saved to: $COMPARISON_FILE"
fi

# Extract key metrics
echo -e "\n${GREEN}Key Performance Metrics:${NC}"
grep -E "Benchmark.*ns/op" "$RESULTS_FILE" | awk '{print $1, $3, $4}' | column -t

# Clean up profile files
rm -f cpu.prof mem.prof