#!/bin/bash

# Real-time Monitoring Script for Pressure Testing
# Usage: ./monitor.sh [output_dir]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
TRACE_DIR="/tmp/pressure-test-traces"
OUTPUT_DIR=${1:-"/tmp/pressure-test-logs"}
REFRESH_INTERVAL=2
MONITOR_LOG="$OUTPUT_DIR/monitor.log"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Initialize log file
echo "Monitoring started at $(date)" > "$MONITOR_LOG"

# Function to get current QPS
get_qps() {
    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        # Count requests in the last 5 seconds
        local now=$(date +%s)
        local five_sec_ago=$((now - 5))

        cat "$TRACE_DIR"/*.jsonl 2>/dev/null | \
            jq -r '.timestamp' 2>/dev/null | \
            while read timestamp; do
                if [ -n "$timestamp" ]; then
                    ts=$(date -d "$timestamp" +%s 2>/dev/null || echo 0)
                    if [ "$ts" -gt "$five_sec_ago" ]; then
                        echo 1
                    fi
                fi
            done | wc -l
    else
        echo 0
    fi
}

# Function to get response time distribution
get_response_distribution() {
    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        cat "$TRACE_DIR"/*.jsonl 2>/dev/null | \
            jq '.duration_ms' 2>/dev/null | \
            awk '
            BEGIN { count[1]=0; count[2]=0; count[3]=0; count[4]=0; count[5]=0 }
            {
                if ($1 < 10) count[1]++
                else if ($1 < 50) count[2]++
                else if ($1 < 100) count[3]++
                else if ($1 < 500) count[4]++
                else count[5]++
            }
            END {
                total = count[1] + count[2] + count[3] + count[4] + count[5]
                if (total > 0) {
                    printf "<10ms:%d(%.1f%%) 10-50ms:%d(%.1f%%) 50-100ms:%d(%.1f%%) 100-500ms:%d(%.1f%%) >500ms:%d(%.1f%%)",
                        count[1], count[1]*100/total,
                        count[2], count[2]*100/total,
                        count[3], count[3]*100/total,
                        count[4], count[4]*100/total,
                        count[5], count[5]*100/total
                } else {
                    printf "No data"
                }
            }'
    else
        echo "No data"
    fi
}

# Function to get error rate
get_error_rate() {
    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        cat "$TRACE_DIR"/*.jsonl 2>/dev/null | \
            jq -r '.status' 2>/dev/null | \
            awk '
            BEGIN { total=0; errors=0 }
            { total++; if ($1 == "error") errors++ }
            END {
                if (total > 0) printf "%.2f%% (%d/%d)", errors*100/total, errors, total
                else printf "0.00%% (0/0)"
            }'
    else
        echo "0.00% (0/0)"
    fi
}

# Function to get top slow operations
get_slow_operations() {
    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        cat "$TRACE_DIR"/*.jsonl 2>/dev/null | \
            jq -r '[.service, .operation, .duration_ms] | @tsv' 2>/dev/null | \
            sort -k3 -rn | \
            head -3 | \
            awk '{printf "  %s.%s: %dms\n", $1, $2, $3}'
    else
        echo "  No data"
    fi
}

# Function to get current memory usage
get_memory_usage() {
    if command -v free &> /dev/null; then
        free -m | awk 'NR==2{printf "%.1fGB used (%.1f%%)", $3/1024, $3*100/$2}'
    else
        echo "N/A"
    fi
}

# Function to get CPU usage
get_cpu_usage() {
    if command -v top &> /dev/null; then
        top -bn1 | grep "Cpu(s)" | awk '{print $2}' | sed 's/%us,//' | head -1
    else
        echo "N/A"
    fi
}

# Main monitoring loop
echo -e "${BLUE}=== Real-time Pressure Test Monitor ===${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop monitoring${NC}"
echo

# Initialize counters
loop_count=0

while true; do
    clear

    # Header
    echo -e "${BLUE}╭─────────────────────────────────────────────────────────────────────────────╮${NC}"
    echo -e "${BLUE}│                    PRESSURE TEST REAL-TIME MONITOR                         │${NC}"
    echo -e "${BLUE}│                           $(date +'%Y-%m-%d %H:%M:%S')                             │${NC}"
    echo -e "${BLUE}╰─────────────────────────────────────────────────────────────────────────────╯${NC}"
    echo

    # System metrics
    echo -e "${CYAN}┌─ System Metrics ─────────────────────────────────────────────┐${NC}"
    printf "│ %-15s: %-40s │\n" "CPU Usage" "$(get_cpu_usage)%"
    printf "│ %-15s: %-40s │\n" "Memory Usage" "$(get_memory_usage)"
    printf "│ %-15s: %-40s │\n" "Monitor Time" "$(date +'%H:%M:%S')"
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo

    # Performance metrics
    echo -e "${GREEN}┌─ Performance Metrics ────────────────────────────────────────┐${NC}"

    # Get current metrics
    current_qps=$(get_qps)
    current_distribution=$(get_response_distribution)
    current_error_rate=$(get_error_rate)

    printf "│ %-15s: %-40s │\n" "Current QPS" "$current_qps req/sec"
    printf "│ %-15s: %-40s │\n" "Error Rate" "$current_error_rate"
    echo -e "${GREEN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo

    # Response time distribution
    echo -e "${YELLOW}┌─ Response Time Distribution ──────────────────────────────────┐${NC}"
    echo "│ $current_distribution"
    echo -e "${YELLOW}└───────────────────────────────────────────────────────────────┘${NC}"
    echo

    # Top slow operations
    echo -e "${RED}┌─ Top 3 Slowest Operations ───────────────────────────────────┐${NC}"
    get_slow_operations
    echo -e "${RED}└───────────────────────────────────────────────────────────────┘${NC}"
    echo

    # Recent activity
    echo -e "${CYAN}┌─ Recent Activity ─────────────────────────────────────────────┐${NC}"
    if [ -f "$TRACE_DIR"/*.jsonl ]; then
        # Show last 3 trace records
        tail -3 "$TRACE_DIR"/*.jsonl 2>/dev/null | \
            jq -r '"\(.timestamp) \(.service).\(.operation) \(.duration_ms)ms \(.status)"' 2>/dev/null | \
            while read line; do
                if [ -n "$line" ]; then
                    echo "│ $line"
                fi
            done
    else
        echo "│ No trace data available yet..."
    fi
    echo -e "${CYAN}└───────────────────────────────────────────────────────────────┘${NC}"
    echo

    # Statistics summary
    if [ -f "$TRACE_DIR"/*.jsonl ] && [ -s "$TRACE_DIR"/*.jsonl ]; then
        echo -e "${BLUE}┌─ Overall Statistics ──────────────────────────────────────────┐${NC}"

        # Calculate overall stats
        total_requests=$(cat "$TRACE_DIR"/*.jsonl 2>/dev/null | wc -l)
        if [ "$total_requests" -gt 0 ]; then
            avg_response=$(cat "$TRACE_DIR"/*.jsonl 2>/dev/null | \
                jq '.duration_ms' 2>/dev/null | \
                awk '{sum+=$1; n++} END {if(n>0) printf "%.1f", sum/n; else print "0"}')

            printf "│ %-15s: %-40s │\n" "Total Requests" "$total_requests"
            printf "│ %-15s: %-40s │\n" "Avg Response" "${avg_response}ms"

            # P95 calculation
            p95_response=$(cat "$TRACE_DIR"/*.jsonl 2>/dev/null | \
                jq '.duration_ms' 2>/dev/null | \
                sort -n | \
                awk '{a[i++]=$1} END {print a[int(i*0.95)]}')

            printf "│ %-15s: %-40s │\n" "P95 Response" "${p95_response}ms"
        else
            printf "│ %-15s: %-40s │\n" "Status" "Waiting for data..."
        fi

        echo -e "${BLUE}└───────────────────────────────────────────────────────────────┘${NC}"
    fi

    # Log monitoring data
    {
        echo "$(date +'%Y-%m-%d %H:%M:%S') - QPS: $current_qps, Error Rate: $current_error_rate"
    } >> "$MONITOR_LOG"

    # Instructions
    echo -e "${YELLOW}Press Ctrl+C to stop monitoring | Refresh every ${REFRESH_INTERVAL}s${NC}"

    # Increment loop counter
    loop_count=$((loop_count + 1))

    # Sleep for refresh interval
    sleep $REFRESH_INTERVAL
done