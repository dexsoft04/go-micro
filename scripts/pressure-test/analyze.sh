#!/bin/bash

# Pressure Test Analysis Script
# Usage: ./analyze.sh [test_session_dir]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
TRACE_DIR="/tmp/pressure-test-traces"
LOG_DIR="/tmp/pressure-test-logs"
DEFAULT_SESSION_DIR="$LOG_DIR/$(ls -t "$LOG_DIR" | grep "test-" | head -1 2>/dev/null || echo "latest")"
SESSION_DIR=${1:-"$DEFAULT_SESSION_DIR"}

echo -e "${BLUE}=== Pressure Test Analysis ===${NC}"

# Check if session directory exists
if [ ! -d "$SESSION_DIR" ]; then
    echo -e "${RED}Error: Session directory not found: $SESSION_DIR${NC}"
    echo "Available sessions:"
    ls -la "$LOG_DIR" | grep "test-" | awk '{print "  " $9}' || echo "  No sessions found"
    exit 1
fi

echo "Analyzing session: $(basename "$SESSION_DIR")"
echo "Session directory: $SESSION_DIR"
echo

# Function to analyze load test results
analyze_load_test() {
    echo -e "${CYAN}=== Load Test Results Analysis ===${NC}"

    # Check for different load test outputs
    if [ -f "$SESSION_DIR/wrk-output.log" ]; then
        echo "Load testing tool: wrk"
        echo

        # Extract key metrics from wrk output
        echo "Key Metrics:"
        grep -E "(Requests/sec|Transfer/sec|Latency|requests in|errors)" "$SESSION_DIR/wrk-output.log" | \
            sed 's/^/  /'

        # Extract percentiles if available
        echo
        echo "Latency Distribution:"
        grep -A 10 "Latency Distribution" "$SESSION_DIR/wrk-output.log" | \
            sed 's/^/  /' || echo "  Not available"

    elif [ -f "$SESSION_DIR/ab-output.log" ]; then
        echo "Load testing tool: Apache Bench"
        echo

        # Extract key metrics from ab output
        echo "Key Metrics:"
        grep -E "(Requests per second|Time per request|Transfer rate|Complete requests|Failed requests)" \
            "$SESSION_DIR/ab-output.log" | sed 's/^/  /'

        echo
        echo "Percentile Distribution:"
        grep -A 20 "Percentage of the requests served within" "$SESSION_DIR/ab-output.log" | \
            sed 's/^/  /' || echo "  Not available"

    elif [ -f "$SESSION_DIR/custom-output.log" ]; then
        echo "Load testing tool: Custom Go client"
        echo

        grep -E "(Total requests|QPS|Error rate|Duration)" "$SESSION_DIR/custom-output.log" | \
            sed 's/^/  /'

    else
        echo -e "${YELLOW}Warning: No load test output found${NC}"
    fi
    echo
}

# Function to analyze trace data
analyze_trace_data() {
    echo -e "${CYAN}=== Trace Data Analysis ===${NC}"

    # Check if trace data exists
    if [ ! -f "$TRACE_DIR"/*.jsonl ] || [ ! -s "$TRACE_DIR"/*.jsonl ]; then
        echo -e "${YELLOW}Warning: No trace data found in $TRACE_DIR${NC}"
        return
    fi

    local trace_files=("$TRACE_DIR"/*.jsonl)
    local total_traces=0

    echo "Trace files found:"
    for file in "${trace_files[@]}"; do
        if [ -f "$file" ]; then
            local count=$(wc -l < "$file")
            total_traces=$((total_traces + count))
            echo "  $(basename "$file"): $count traces"
        fi
    done

    echo "Total traces: $total_traces"
    echo

    if [ $total_traces -eq 0 ]; then
        echo -e "${YELLOW}No trace data to analyze${NC}"
        return
    fi

    # Run detailed trace analysis
    echo "Generating detailed trace analysis..."

    # Use trace-analyzer if available
    if [ -f "../../cmd/trace-analyzer/main.go" ]; then
        echo
        echo -e "${GREEN}=== Service Performance Summary ===${NC}"
        cat "$TRACE_DIR"/*.jsonl | go run ../../cmd/trace-analyzer/main.go -stream -top 10

        # Generate different analysis views
        echo
        echo -e "${GREEN}=== Slow Operations (>100ms) ===${NC}"
        cat "$TRACE_DIR"/*.jsonl | go run ../../cmd/trace-analyzer/main.go -stream -min-duration 100 -top 5

        echo
        echo -e "${GREEN}=== Error Analysis ===${NC}"
        cat "$TRACE_DIR"/*.jsonl | go run ../../cmd/trace-analyzer/main.go -stream -errors

        # Save detailed HTML report
        echo
        echo "Generating HTML report..."
        local html_report="$SESSION_DIR/detailed-trace-analysis.html"
        cat "$TRACE_DIR"/*.jsonl | \
            go run ../../cmd/trace-analyzer/main.go -format html -output "$html_report" 2>/dev/null && \
            echo "HTML report saved: $html_report"

    else
        # Fallback analysis using jq and awk
        echo "Using fallback analysis (trace-analyzer not available)"

        echo
        echo "Performance Summary:"
        cat "$TRACE_DIR"/*.jsonl | jq -s '
            group_by(.service + "." + .operation) |
            map({
                operation: (.[0].service + "." + .[0].operation),
                count: length,
                avg_ms: (map(.duration_ms) | add / length),
                max_ms: (map(.duration_ms) | max),
                min_ms: (map(.duration_ms) | min)
            }) |
            sort_by(.avg_ms) | reverse |
            .[] |
            "  \(.operation): \(.count) calls, avg=\(.avg_ms)ms, max=\(.max_ms)ms"
        ' | head -10
    fi
}

# Function to generate performance insights
generate_insights() {
    echo -e "${CYAN}=== Performance Insights ===${NC}"

    local insights_count=0

    # Check for high error rates
    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        local error_rate=$(cat "$TRACE_DIR"/*.jsonl | \
            jq -r '.status' | \
            awk 'BEGIN{total=0; errors=0} {total++; if($1=="error") errors++} END{if(total>0) print errors*100/total; else print 0}')

        if (( $(echo "$error_rate > 1.0" | bc -l) )); then
            echo -e "${RED}⚠ High Error Rate Detected: ${error_rate}%${NC}"
            insights_count=$((insights_count + 1))
        fi

        # Check for slow operations
        local slow_count=$(cat "$TRACE_DIR"/*.jsonl | \
            jq '.duration_ms' | \
            awk '{if($1>1000) count++} END{print count+0}')

        if [ "$slow_count" -gt 0 ]; then
            echo -e "${YELLOW}⚠ Slow Operations Detected: $slow_count operations >1000ms${NC}"
            insights_count=$((insights_count + 1))
        fi

        # Check for performance degradation over time
        local first_half_avg=$(cat "$TRACE_DIR"/*.jsonl | \
            head -n $((total_traces / 2)) | \
            jq '.duration_ms' | \
            awk '{sum+=$1; n++} END{if(n>0) print sum/n; else print 0}')

        local second_half_avg=$(cat "$TRACE_DIR"/*.jsonl | \
            tail -n $((total_traces / 2)) | \
            jq '.duration_ms' | \
            awk '{sum+=$1; n++} END{if(n>0) print sum/n; else print 0}')

        if (( $(echo "$second_half_avg > $first_half_avg * 1.5" | bc -l) )); then
            echo -e "${YELLOW}⚠ Performance Degradation: Response time increased from ${first_half_avg}ms to ${second_half_avg}ms${NC}"
            insights_count=$((insights_count + 1))
        fi

        # Check for memory leaks (if monitoring data available)
        if [ -f "$SESSION_DIR/monitor.log" ]; then
            local memory_trend=$(tail -20 "$SESSION_DIR/monitor.log" | \
                grep -o "Memory:.*%" | \
                awk '{print $2}' | \
                sed 's/%//' | \
                awk 'BEGIN{sum=0; n=0} {sum+=$1; n++} END{if(n>0) print sum/n}')

            if (( $(echo "$memory_trend > 80" | bc -l) )); then
                echo -e "${RED}⚠ High Memory Usage: ${memory_trend}%${NC}"
                insights_count=$((insights_count + 1))
            fi
        fi
    fi

    if [ $insights_count -eq 0 ]; then
        echo -e "${GREEN}✓ No critical performance issues detected${NC}"
    fi

    echo
}

# Function to generate recommendations
generate_recommendations() {
    echo -e "${CYAN}=== Optimization Recommendations ===${NC}"

    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        # Analyze response time distribution
        local p95=$(cat "$TRACE_DIR"/*.jsonl | \
            jq '.duration_ms' | \
            sort -n | \
            awk '{a[i++]=$1} END{print a[int(i*0.95)]}')

        local p99=$(cat "$TRACE_DIR"/*.jsonl | \
            jq '.duration_ms' | \
            sort -n | \
            awk '{a[i++]=$1} END{print a[int(i*0.99)]}')

        echo "Response Time Analysis:"
        echo "  P95: ${p95}ms"
        echo "  P99: ${p99}ms"
        echo

        # Generate recommendations based on thresholds
        if (( $(echo "$p95 > 200" | bc -l) )); then
            echo "📋 Recommendations for P95 > 200ms:"
            echo "  • Review database query performance"
            echo "  • Consider implementing caching"
            echo "  • Check for N+1 query problems"
            echo "  • Review API endpoint efficiency"
            echo
        fi

        if (( $(echo "$p99 > 1000" | bc -l) )); then
            echo "📋 Recommendations for P99 > 1000ms:"
            echo "  • Implement request timeouts"
            echo "  • Add circuit breakers for external dependencies"
            echo "  • Review resource allocation"
            echo "  • Consider horizontal scaling"
            echo
        fi

        # Service-specific recommendations
        echo "Service-specific Analysis:"
        cat "$TRACE_DIR"/*.jsonl | \
            jq -r '.service' | \
            sort | uniq -c | sort -rn | \
            head -5 | \
            while read count service; do
                local service_avg=$(cat "$TRACE_DIR"/*.jsonl | \
                    jq "select(.service==\"$service\") | .duration_ms" | \
                    awk '{sum+=$1; n++} END{if(n>0) print sum/n; else print 0}')
                echo "  $service: $count requests, avg=${service_avg}ms"
            done
    fi

    echo
}

# Function to compare with thresholds
compare_with_thresholds() {
    echo -e "${CYAN}=== Threshold Comparison ===${NC}"

    local config_file="$(dirname "$0")/configs/thresholds.yaml"

    if [ -f "$config_file" ] && command -v yq &> /dev/null; then
        local p50_threshold=$(yq eval '.thresholds.response_time.p50' "$config_file")
        local p95_threshold=$(yq eval '.thresholds.response_time.p95' "$config_file")
        local p99_threshold=$(yq eval '.thresholds.response_time.p99' "$config_file")
        local error_threshold=$(yq eval '.thresholds.error_rate.max' "$config_file")

        echo "Configured Thresholds:"
        echo "  P50: ${p50_threshold}ms"
        echo "  P95: ${p95_threshold}ms"
        echo "  P99: ${p99_threshold}ms"
        echo "  Max Error Rate: $(echo "$error_threshold * 100" | bc)%"
        echo

        # Compare actual vs thresholds
        if [ -f "$TRACE_DIR"/*.jsonl ]; then
            echo "Threshold Compliance:"
            # Add threshold comparison logic here
            echo "  (Comparison logic to be implemented)"
        fi
    else
        echo "Threshold configuration not available"
    fi

    echo
}

# Main analysis execution
echo "Starting analysis..."
echo

# Run all analysis functions
analyze_load_test
analyze_trace_data
generate_insights
generate_recommendations
compare_with_thresholds

# Generate final summary
echo -e "${BLUE}=== Analysis Summary ===${NC}"
echo "Session: $(basename "$SESSION_DIR")"
echo "Analysis completed at: $(date)"

# List generated files
echo
echo "Generated Files:"
find "$SESSION_DIR" -type f -exec basename {} \; | sort | sed 's/^/  - /'

# Suggest next steps
echo
echo -e "${GREEN}=== Next Steps ===${NC}"
echo "1. Review HTML report (if generated)"
echo "2. Compare results with previous tests"
echo "3. Implement recommended optimizations"
echo "4. Run regression tests to validate improvements"

# Save analysis metadata
cat > "$SESSION_DIR/analysis-metadata.json" << EOF
{
  "analysis_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "trace_files_analyzed": $(find "$TRACE_DIR" -name "*.jsonl" -type f | wc -l),
  "total_traces": $(cat "$TRACE_DIR"/*.jsonl 2>/dev/null | wc -l || echo 0),
  "session_directory": "$SESSION_DIR"
}
EOF

echo
echo -e "${GREEN}Analysis completed successfully!${NC}"