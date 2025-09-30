#!/bin/bash

# Pressure Test Setup Script
# This script prepares the environment for pressure testing with trace collection

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TRACE_DIR="/tmp/pressure-test-traces"
CONFIG_DIR="$(dirname "$0")/configs"
LOG_DIR="/tmp/pressure-test-logs"

echo -e "${BLUE}=== Pressure Test Environment Setup ===${NC}"

# Create directories
echo -e "${YELLOW}1. Creating directories...${NC}"
mkdir -p "$TRACE_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "$CONFIG_DIR"

# Clear existing data
echo -e "${YELLOW}2. Cleaning previous test data...${NC}"
rm -f "$TRACE_DIR"/*.jsonl
rm -f "$LOG_DIR"/*.log

# Check dependencies
echo -e "${YELLOW}3. Checking dependencies...${NC}"

# Check if go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

# Check if trace-analyzer can be built
if ! go build -o /tmp/trace-analyzer ./cmd/trace-analyzer/main.go; then
    echo -e "${RED}Error: Cannot build trace-analyzer${NC}"
    exit 1
fi

# Check for optional tools
echo "Checking optional tools:"
for tool in wrk ab jq curl; do
    if command -v $tool &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} $tool"
    else
        echo -e "  ${YELLOW}!${NC} $tool (optional)"
    fi
done

# Generate default configuration
echo -e "${YELLOW}4. Generating default configuration...${NC}"

cat > "$CONFIG_DIR/services.yaml" << 'EOF'
# Service configuration for pressure testing
services:
  user-service:
    instances: 2
    port_range: "8001-8002"
    endpoints:
      - "/api/users"
      - "/api/users/{id}"

  order-service:
    instances: 2
    port_range: "8011-8012"
    endpoints:
      - "/api/orders"
      - "/api/orders/{id}"

  payment-service:
    instances: 1
    port_range: "8021"
    endpoints:
      - "/api/payments"
EOF

cat > "$CONFIG_DIR/scenarios.yaml" << 'EOF'
# Pressure test scenarios
scenarios:
  light:
    duration: "30s"
    threads: 4
    connections: 50
    description: "Light load testing"

  medium:
    duration: "60s"
    threads: 8
    connections: 200
    description: "Medium load testing"

  heavy:
    duration: "120s"
    threads: 12
    connections: 500
    description: "Heavy load testing"

  stress:
    duration: "300s"
    threads: 16
    connections: 1000
    description: "Stress testing"
EOF

cat > "$CONFIG_DIR/thresholds.yaml" << 'EOF'
# Performance thresholds
thresholds:
  response_time:
    p50: 50   # ms
    p95: 200  # ms
    p99: 500  # ms

  error_rate:
    max: 0.01  # 1%

  throughput:
    min_qps: 100
EOF

# Set environment variables for pressure testing
echo -e "${YELLOW}5. Setting up environment variables...${NC}"

cat > "$TRACE_DIR/pressure-test.env" << EOF
# Pressure testing environment variables
export MICRO_PRESSURE_TEST_MODE=true
export MICRO_TRACING_FILE_FORMAT=simple
export MICRO_INSTANCE_ID=\${HOSTNAME}-\${SERVICE_NAME}

# Trace output directory (will be set per service)
export TRACE_OUTPUT_DIR=$TRACE_DIR

# Log configuration
export LOG_OUTPUT_DIR=$LOG_DIR
EOF

# Create convenience scripts
echo -e "${YELLOW}6. Creating convenience scripts...${NC}"

# Source environment helper
cat > "$TRACE_DIR/load-env.sh" << 'EOF'
#!/bin/bash
# Load pressure testing environment

SERVICE_NAME=${1:-"test-service"}
INSTANCE_ID=${2:-"${HOSTNAME}-${SERVICE_NAME}"}

export MICRO_PRESSURE_TEST_MODE=true
export MICRO_SERVICE_NAME="$SERVICE_NAME"
export MICRO_INSTANCE_ID="$INSTANCE_ID"
export MICRO_TRACING_REPORTER_ADDRESS="file:///tmp/pressure-test-traces/${SERVICE_NAME}-${INSTANCE_ID}.jsonl"
export MICRO_TRACING_FILE_FORMAT=simple

echo "Environment loaded for service: $SERVICE_NAME"
echo "Trace output: $MICRO_TRACING_REPORTER_ADDRESS"
EOF

chmod +x "$TRACE_DIR/load-env.sh"

# Create test data generator
cat > "$TRACE_DIR/generate-test-data.sh" << 'EOF'
#!/bin/bash
# Generate sample trace data for testing

TRACE_FILE=${1:-"/tmp/pressure-test-traces/sample-traces.jsonl"}

echo "Generating sample trace data to: $TRACE_FILE"

# Generate 1000 sample trace records
for i in {1..1000}; do
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")
    service="service-$((RANDOM % 3 + 1))"
    operation="operation-$((RANDOM % 5 + 1))"
    duration=$((RANDOM % 500 + 10))
    status=$( [ $((RANDOM % 100)) -lt 95 ] && echo "ok" || echo "error" )
    trace_id=$(printf "%032d" $i)
    span_id=$(printf "%016d" $i)

    cat >> "$TRACE_FILE" << EOL
{"timestamp":"$timestamp","service":"$service","operation":"$operation","duration_ms":$duration,"trace_id":"$trace_id","span_id":"$span_id","status":"$status","attributes":{"http.method":"GET"}}
EOL
done

echo "Generated 1000 sample trace records"
EOF

chmod +x "$TRACE_DIR/generate-test-data.sh"

# Print setup summary
echo -e "${GREEN}=== Setup Complete ===${NC}"
echo
echo -e "${BLUE}Directories created:${NC}"
echo "  - Trace output: $TRACE_DIR"
echo "  - Logs: $LOG_DIR"
echo "  - Config: $CONFIG_DIR"
echo
echo -e "${BLUE}Configuration files:${NC}"
echo "  - services.yaml: Service definitions"
echo "  - scenarios.yaml: Test scenarios"
echo "  - thresholds.yaml: Performance thresholds"
echo
echo -e "${BLUE}Environment:${NC}"
echo "  - Load env: source $TRACE_DIR/load-env.sh <service-name>"
echo "  - Test data: $TRACE_DIR/generate-test-data.sh"
echo
echo -e "${BLUE}Next steps:${NC}"
echo "  1. Source environment for your services"
echo "  2. Run: ./run-test.sh <scenario>"
echo "  3. Monitor: ./monitor.sh"
echo "  4. Analyze: ./analyze.sh"
echo
echo -e "${GREEN}Ready for pressure testing!${NC}"