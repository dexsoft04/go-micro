#!/bin/bash

# Pressure Test Execution Script
# Usage: ./run-test.sh [scenario] [target_url]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default configuration
SCRIPT_DIR="$(dirname "$0")"
TRACE_DIR="/tmp/pressure-test-traces"
LOG_DIR="/tmp/pressure-test-logs"
CONFIG_DIR="$SCRIPT_DIR/configs"

# Parse command line arguments
SCENARIO=${1:-"medium"}
TARGET_URL=${2:-"http://localhost:8080"}
CUSTOM_DURATION=$3
CUSTOM_THREADS=$4
CUSTOM_CONNECTIONS=$5

# Load scenario configuration
if [ -f "$CONFIG_DIR/scenarios.yaml" ]; then
    if command -v yq &> /dev/null; then
        DURATION=$(yq eval ".scenarios.$SCENARIO.duration" "$CONFIG_DIR/scenarios.yaml" 2>/dev/null || echo "60s")
        THREADS=$(yq eval ".scenarios.$SCENARIO.threads" "$CONFIG_DIR/scenarios.yaml" 2>/dev/null || echo "8")
        CONNECTIONS=$(yq eval ".scenarios.$SCENARIO.connections" "$CONFIG_DIR/scenarios.yaml" 2>/dev/null || echo "200")
        DESCRIPTION=$(yq eval ".scenarios.$SCENARIO.description" "$CONFIG_DIR/scenarios.yaml" 2>/dev/null || echo "Load test")
    else
        # Default values if yq is not available
        case $SCENARIO in
            "light")
                DURATION="30s"; THREADS=4; CONNECTIONS=50; DESCRIPTION="Light load testing"
                ;;
            "medium")
                DURATION="60s"; THREADS=8; CONNECTIONS=200; DESCRIPTION="Medium load testing"
                ;;
            "heavy")
                DURATION="120s"; THREADS=12; CONNECTIONS=500; DESCRIPTION="Heavy load testing"
                ;;
            "stress")
                DURATION="300s"; THREADS=16; CONNECTIONS=1000; DESCRIPTION="Stress testing"
                ;;
            *)
                DURATION="60s"; THREADS=8; CONNECTIONS=200; DESCRIPTION="Custom test"
                ;;
        esac
    fi
else
    # Default values
    DURATION="60s"; THREADS=8; CONNECTIONS=200; DESCRIPTION="Default test"
fi

# Override with custom values if provided
[ -n "$CUSTOM_DURATION" ] && DURATION="$CUSTOM_DURATION"
[ -n "$CUSTOM_THREADS" ] && THREADS="$CUSTOM_THREADS"
[ -n "$CUSTOM_CONNECTIONS" ] && CONNECTIONS="$CUSTOM_CONNECTIONS"

# Create test session directory
TEST_SESSION="test-$(date +%Y%m%d-%H%M%S)"
SESSION_DIR="$LOG_DIR/$TEST_SESSION"
mkdir -p "$SESSION_DIR"

echo -e "${BLUE}=== Pressure Test Execution ===${NC}"
echo "Scenario: $SCENARIO ($DESCRIPTION)"
echo "Target: $TARGET_URL"
echo "Duration: $DURATION"
echo "Threads: $THREADS"
echo "Connections: $CONNECTIONS"
echo "Session: $TEST_SESSION"
echo

# Check if target is reachable
echo -e "${YELLOW}Checking target availability...${NC}"
if ! curl -s --connect-timeout 5 "$TARGET_URL/health" > /dev/null 2>&1; then
    echo -e "${YELLOW}Warning: Health check failed, but continuing...${NC}"
fi

# Clear previous trace data
echo -e "${YELLOW}Clearing previous trace data...${NC}"
rm -f "$TRACE_DIR"/*.jsonl

# Start monitoring in background
echo -e "${YELLOW}Starting monitoring...${NC}"
"$SCRIPT_DIR/monitor.sh" "$SESSION_DIR" &
MONITOR_PID=$!

# Wait a moment for monitoring to start
sleep 2

# Record test start time
TEST_START=$(date +%s)
echo "Test started at: $(date)"

# Create test report header
cat > "$SESSION_DIR/test-report.md" << EOF
# Pressure Test Report

## Test Configuration
- **Scenario**: $SCENARIO
- **Description**: $DESCRIPTION
- **Target URL**: $TARGET_URL
- **Duration**: $DURATION
- **Threads**: $THREADS
- **Connections**: $CONNECTIONS
- **Start Time**: $(date)
- **Session ID**: $TEST_SESSION

## Test Progress
EOF

# Function to run wrk if available
run_wrk_test() {
    echo -e "${GREEN}Running wrk test...${NC}"

    # Convert duration to seconds for wrk
    DURATION_SEC=$(echo "$DURATION" | sed 's/[^0-9]//g')

    wrk -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" \
        --latency \
        --timeout 30s \
        --script <(cat << 'EOF'
-- wrk Lua script for detailed reporting
local counter = 0
local start_time = os.time()

request = function()
    counter = counter + 1
    return wrk.request()
end

response = function(status, headers, body)
    if status ~= 200 then
        print(string.format("Non-200 response: %d at request %d", status, counter))
    end
end

done = function(summary, latency, requests)
    local end_time = os.time()
    print(string.format("\n=== WRK Test Summary ==="))
    print(string.format("Duration: %d seconds", end_time - start_time))
    print(string.format("Requests: %d", summary.requests))
    print(string.format("Bytes transferred: %d", summary.bytes))
    print(string.format("Errors (connect/read/write/status): %d/%d/%d/%d",
        summary.errors.connect, summary.errors.read, summary.errors.write, summary.errors.status))
    print(string.format("QPS: %.2f", summary.requests / (end_time - start_time)))
    print(string.format("Latency min/max/mean/stdev: %.2fms/%.2fms/%.2fms/%.2fms",
        latency.min / 1000, latency.max / 1000, latency.mean / 1000, latency.stdev / 1000))
    print(string.format("Latency percentiles 50th/90th/95th/99th: %.2fms/%.2fms/%.2fms/%.2fms",
        latency:percentile(50) / 1000, latency:percentile(90) / 1000,
        latency:percentile(95) / 1000, latency:percentile(99) / 1000))
end
EOF
        ) \
        "$TARGET_URL" 2>&1 | tee "$SESSION_DIR/wrk-output.log"
}

# Function to run Apache Bench if wrk is not available
run_ab_test() {
    echo -e "${GREEN}Running Apache Bench test...${NC}"

    # Convert duration to number of requests (estimate)
    DURATION_SEC=$(echo "$DURATION" | sed 's/[^0-9]//g')
    TOTAL_REQUESTS=$((DURATION_SEC * CONNECTIONS / 2))  # Conservative estimate

    ab -n "$TOTAL_REQUESTS" -c "$CONNECTIONS" -g "$SESSION_DIR/ab-gnuplot.dat" \
        "$TARGET_URL/" 2>&1 | tee "$SESSION_DIR/ab-output.log"
}

# Function to run custom Go client
run_custom_test() {
    echo -e "${GREEN}Running custom test...${NC}"

    # Create a simple Go test client
    cat > "$SESSION_DIR/custom-client.go" << 'EOF'
package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
    "os"
    "strconv"
    "context"
)

func main() {
    if len(os.Args) < 5 {
        fmt.Println("Usage: go run custom-client.go <url> <duration> <threads> <connections>")
        os.Exit(1)
    }

    url := os.Args[1]
    duration, _ := time.ParseDuration(os.Args[2])
    threads, _ := strconv.Atoi(os.Args[3])
    connections, _ := strconv.Atoi(os.Args[4])

    fmt.Printf("Starting custom test: %s for %v with %d threads and %d connections\n",
        url, duration, threads, connections)

    var wg sync.WaitGroup
    requestCount := int64(0)
    errorCount := int64(0)

    start := time.Now()
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()

    for i := 0; i < threads; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            client := &http.Client{Timeout: 30 * time.Second}

            for {
                select {
                case <-ctx.Done():
                    return
                default:
                    resp, err := client.Get(url)
                    requestCount++
                    if err != nil || resp.StatusCode != 200 {
                        errorCount++
                    }
                    if resp != nil {
                        resp.Body.Close()
                    }
                    time.Sleep(time.Millisecond * 10) // Small delay
                }
            }
        }()
    }

    wg.Wait()
    elapsed := time.Since(start)

    fmt.Printf("\n=== Custom Test Results ===\n")
    fmt.Printf("Total requests: %d\n", requestCount)
    fmt.Printf("Errors: %d\n", errorCount)
    fmt.Printf("Duration: %v\n", elapsed)
    fmt.Printf("QPS: %.2f\n", float64(requestCount)/elapsed.Seconds())
    fmt.Printf("Error rate: %.2f%%\n", float64(errorCount)/float64(requestCount)*100)
}
EOF

    go run "$SESSION_DIR/custom-client.go" "$TARGET_URL" "$DURATION" "$THREADS" "$CONNECTIONS" \
        2>&1 | tee "$SESSION_DIR/custom-output.log"
}

# Choose test tool and run
if command -v wrk &> /dev/null; then
    run_wrk_test
elif command -v ab &> /dev/null; then
    run_ab_test
else
    echo -e "${YELLOW}Neither wrk nor ab found, using custom Go client...${NC}"
    run_custom_test
fi

# Record test end time
TEST_END=$(date +%s)
TEST_DURATION=$((TEST_END - TEST_START))

echo
echo -e "${GREEN}Test completed in ${TEST_DURATION} seconds${NC}"

# Stop monitoring
kill $MONITOR_PID 2>/dev/null || true
sleep 2

# Generate analysis report
echo -e "${YELLOW}Generating analysis report...${NC}"
if [ -f "$TRACE_DIR"/*.jsonl ] && [ -s "$TRACE_DIR"/*.jsonl ]; then
    # Run trace analysis
    cat "$TRACE_DIR"/*.jsonl | go run ../../cmd/trace-analyzer/main.go -stream > "$SESSION_DIR/trace-analysis.txt" 2>/dev/null || true

    # Generate HTML report
    cat "$TRACE_DIR"/*.jsonl | go run ../../cmd/trace-analyzer/main.go -format html -output "$SESSION_DIR/trace-report.html" 2>/dev/null || true

    echo "Trace analysis completed"
else
    echo -e "${YELLOW}Warning: No trace data found${NC}"
fi

# Update test report
cat >> "$SESSION_DIR/test-report.md" << EOF

## Test Results
- **End Time**: $(date)
- **Actual Duration**: ${TEST_DURATION} seconds
- **Test Tool**: $(command -v wrk &> /dev/null && echo "wrk" || (command -v ab &> /dev/null && echo "ab" || echo "custom"))

## Files Generated
- \`wrk-output.log\` or \`ab-output.log\` or \`custom-output.log\`: Load test results
- \`trace-analysis.txt\`: Trace performance analysis
- \`trace-report.html\`: HTML trace report
- \`monitor.log\`: System monitoring log

EOF

# Show summary
echo
echo -e "${BLUE}=== Test Session Summary ===${NC}"
echo "Session directory: $SESSION_DIR"
echo "Files generated:"
find "$SESSION_DIR" -type f -exec basename {} \; | sort | sed 's/^/  - /'

if [ -f "$SESSION_DIR/trace-report.html" ]; then
    echo
    echo -e "${GREEN}HTML report available: $SESSION_DIR/trace-report.html${NC}"
    echo "Open with: xdg-open $SESSION_DIR/trace-report.html"
fi

echo
echo -e "${GREEN}Pressure test completed successfully!${NC}"