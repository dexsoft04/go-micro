package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// TraceRecord represents a trace record from the simple file exporter
type TraceRecord struct {
	Timestamp    string                 `json:"timestamp"`
	Service      string                 `json:"service"`
	Operation    string                 `json:"operation"`
	DurationMs   int64                  `json:"duration_ms"`
	TraceID      string                 `json:"trace_id"`
	SpanID       string                 `json:"span_id"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
	Status       string                 `json:"status"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// PerformanceStats holds performance statistics for an operation
type PerformanceStats struct {
	Operation  string
	Service    string
	Count      int
	TotalMs    int64
	MinMs      int64
	MaxMs      int64
	AvgMs      float64
	P50Ms      int64
	P95Ms      int64
	P99Ms      int64
	ErrorCount int
	ErrorRate  float64
	Durations  []int64
}

func main() {
	var (
		inputFile     = flag.String("input", "", "Input trace file (JSONL format)")
		outputFormat  = flag.String("format", "table", "Output format: table, json, html, pressure")
		outputFile    = flag.String("output", "", "Output file (default: stdout)")
		serviceFilter = flag.String("service", "", "Filter by service name")
		minDuration   = flag.Int64("min-duration", 0, "Filter by minimum duration (ms)")
		showErrors    = flag.Bool("errors", false, "Show only errors")
		streamMode    = flag.Bool("stream", false, "Stream mode: analyze traces in real-time")
		topN          = flag.Int("top", 10, "Show top N slowest operations")
		pressureMode  = flag.Bool("pressure", false, "Pressure testing mode: detailed performance analysis")
		timeWindow    = flag.String("time-window", "", "Time window for analysis (e.g., 30s, 5m)")
		showTrends    = flag.Bool("trends", false, "Show performance trends over time")
		qpsReport     = flag.Bool("qps", false, "Generate QPS report")
	)
	flag.Parse()

	if *inputFile == "" && !*streamMode {
		log.Fatal("Please specify input file with -input or use -stream for real-time analysis")
	}

	var reader io.Reader
	if *streamMode {
		reader = os.Stdin
	} else {
		file, err := os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Failed to open input file: %v", err)
		}
		defer file.Close()
		reader = file
	}

	records, err := parseTraceRecords(reader, *serviceFilter, *minDuration, *showErrors)
	if err != nil {
		log.Fatalf("Failed to parse trace records: %v", err)
	}

	stats := analyzePerformance(records)

	var output io.Writer = os.Stdout
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		output = file
	}

	// Handle pressure testing mode
	if *pressureMode || *outputFormat == "pressure" {
		printPressureTestingReport(output, records, stats, *topN, *qpsReport, *showTrends, *timeWindow)
		return
	}

	switch *outputFormat {
	case "table":
		printTableReport(output, stats, *topN)
	case "json":
		printJSONReport(output, stats)
	case "html":
		printHTMLReport(output, stats, *topN)
	default:
		log.Fatalf("Unknown output format: %s", *outputFormat)
	}
}

func parseTraceRecords(reader io.Reader, serviceFilter string, minDuration int64, showErrors bool) ([]TraceRecord, error) {
	var records []TraceRecord
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record TraceRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			// Skip invalid JSON lines
			continue
		}

		// Apply filters
		if serviceFilter != "" && record.Service != serviceFilter {
			continue
		}
		if minDuration > 0 && record.DurationMs < minDuration {
			continue
		}
		if showErrors && record.Status != "error" {
			continue
		}

		records = append(records, record)
	}

	return records, scanner.Err()
}

func analyzePerformance(records []TraceRecord) map[string]*PerformanceStats {
	statsMap := make(map[string]*PerformanceStats)

	for _, record := range records {
		key := fmt.Sprintf("%s.%s", record.Service, record.Operation)

		stats, exists := statsMap[key]
		if !exists {
			stats = &PerformanceStats{
				Operation: record.Operation,
				Service:   record.Service,
				MinMs:     record.DurationMs,
				MaxMs:     record.DurationMs,
				Durations: make([]int64, 0),
			}
			statsMap[key] = stats
		}

		// Update statistics
		stats.Count++
		stats.TotalMs += record.DurationMs
		stats.Durations = append(stats.Durations, record.DurationMs)

		if record.DurationMs < stats.MinMs {
			stats.MinMs = record.DurationMs
		}
		if record.DurationMs > stats.MaxMs {
			stats.MaxMs = record.DurationMs
		}

		if record.Status == "error" {
			stats.ErrorCount++
		}
	}

	// Calculate percentiles and derived metrics
	for _, stats := range statsMap {
		stats.AvgMs = float64(stats.TotalMs) / float64(stats.Count)
		stats.ErrorRate = float64(stats.ErrorCount) / float64(stats.Count) * 100

		// Sort durations for percentile calculation
		sort.Slice(stats.Durations, func(i, j int) bool {
			return stats.Durations[i] < stats.Durations[j]
		})

		// Calculate percentiles
		if len(stats.Durations) > 0 {
			stats.P50Ms = percentile(stats.Durations, 50)
			stats.P95Ms = percentile(stats.Durations, 95)
			stats.P99Ms = percentile(stats.Durations, 99)
		}
	}

	return statsMap
}

func percentile(sortedData []int64, p int) int64 {
	if len(sortedData) == 0 {
		return 0
	}

	index := float64(p) / 100.0 * float64(len(sortedData)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedData) {
		return sortedData[len(sortedData)-1]
	}

	weight := index - float64(lower)
	return int64(float64(sortedData[lower])*(1-weight) + float64(sortedData[upper])*weight)
}

func printTableReport(w io.Writer, statsMap map[string]*PerformanceStats, topN int) {
	// Convert map to slice for sorting
	var statsList []*PerformanceStats
	for _, stats := range statsMap {
		statsList = append(statsList, stats)
	}

	// Sort by P95 duration (descending)
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].P95Ms > statsList[j].P95Ms
	})

	fmt.Fprintf(w, "=== Performance Analysis Report ===\n")
	fmt.Fprintf(w, "Generated at: %s\n\n", time.Now().Format(time.RFC3339))

	fmt.Fprintf(w, "%-30s %-20s %8s %8s %8s %8s %8s %8s %8s %8s\n",
		"Operation", "Service", "Count", "Avg(ms)", "Min(ms)", "Max(ms)", "P50(ms)", "P95(ms)", "P99(ms)", "Errors(%)")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 150))

	displayCount := topN
	if len(statsList) < topN {
		displayCount = len(statsList)
	}

	for i := 0; i < displayCount; i++ {
		stats := statsList[i]
		fmt.Fprintf(w, "%-30s %-20s %8d %8.1f %8d %8d %8d %8d %8d %8.1f\n",
			truncateString(stats.Operation, 30),
			truncateString(stats.Service, 20),
			stats.Count,
			stats.AvgMs,
			stats.MinMs,
			stats.MaxMs,
			stats.P50Ms,
			stats.P95Ms,
			stats.P99Ms,
			stats.ErrorRate)
	}
}

func printJSONReport(w io.Writer, statsMap map[string]*PerformanceStats) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"statistics":   statsMap,
	})
}

func printHTMLReport(w io.Writer, statsMap map[string]*PerformanceStats, topN int) {
	// Convert map to slice for sorting
	var statsList []*PerformanceStats
	for _, stats := range statsMap {
		statsList = append(statsList, stats)
	}

	// Sort by P95 duration (descending)
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].P95Ms > statsList[j].P95Ms
	})

	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Trace Performance Analysis</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; }
		table { border-collapse: collapse; width: 100%%; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: right; }
		th { background-color: #f2f2f2; text-align: center; }
		.service { text-align: left; }
		.operation { text-align: left; }
		.error-high { background-color: #ffebee; }
		.slow { background-color: #fff3e0; }
	</style>
</head>
<body>
	<h1>Performance Analysis Report</h1>
	<p>Generated at: %s</p>

	<table>
		<tr>
			<th class="operation">Operation</th>
			<th class="service">Service</th>
			<th>Count</th>
			<th>Avg(ms)</th>
			<th>Min(ms)</th>
			<th>Max(ms)</th>
			<th>P50(ms)</th>
			<th>P95(ms)</th>
			<th>P99(ms)</th>
			<th>Errors(%%)</th>
		</tr>
`, time.Now().Format(time.RFC3339))

	displayCount := topN
	if len(statsList) < topN {
		displayCount = len(statsList)
	}

	for i := 0; i < displayCount; i++ {
		stats := statsList[i]

		class := ""
		if stats.ErrorRate > 5 {
			class = "error-high"
		} else if stats.P95Ms > 1000 {
			class = "slow"
		}

		fmt.Fprintf(w, `		<tr class="%s">
			<td class="operation">%s</td>
			<td class="service">%s</td>
			<td>%d</td>
			<td>%.1f</td>
			<td>%d</td>
			<td>%d</td>
			<td>%d</td>
			<td>%d</td>
			<td>%d</td>
			<td>%.1f</td>
		</tr>
`, class, stats.Operation, stats.Service, stats.Count, stats.AvgMs,
			stats.MinMs, stats.MaxMs, stats.P50Ms, stats.P95Ms, stats.P99Ms, stats.ErrorRate)
	}

	fmt.Fprintf(w, `	</table>
</body>
</html>`)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// PressureTestingMetrics holds comprehensive metrics for pressure testing
type PressureTestingMetrics struct {
	TotalRequests    int
	TotalDuration    time.Duration
	OverallQPS       float64
	SuccessRate      float64
	ErrorRate        float64
	TimeWindow       []TimeWindowStats
	ServiceBreakdown map[string]ServiceMetrics
}

type TimeWindowStats struct {
	StartTime    time.Time
	EndTime      time.Time
	RequestCount int
	QPS          float64
	AvgLatency   float64
	ErrorCount   int
}

type ServiceMetrics struct {
	RequestCount int
	AvgLatency   float64
	P99Latency   float64
	ErrorRate    float64
	QPS          float64
}

func printPressureTestingReport(w io.Writer, records []TraceRecord, statsMap map[string]*PerformanceStats, topN int, qpsReport, showTrends bool, timeWindow string) {
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "                         PRESSURE TESTING ANALYSIS REPORT                     \n")
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Total trace records analyzed: %d\n\n", len(records))

	// Overall performance summary
	printOverallSummary(w, records, statsMap)

	// Top problematic operations
	fmt.Fprintf(w, "\n┌─ TOP %d SLOWEST OPERATIONS ──────────────────────────────────────────────┐\n", topN)
	printTopSlowOperations(w, statsMap, topN)
	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")

	// Error analysis
	printErrorAnalysis(w, records, statsMap)

	// QPS analysis
	if qpsReport {
		printQPSAnalysis(w, records, timeWindow)
	}

	// Performance trends
	if showTrends {
		printPerformanceTrends(w, records, timeWindow)
	}

	// Service breakdown
	printServiceBreakdown(w, records, statsMap)

	// Optimization recommendations
	printOptimizationRecommendations(w, statsMap, records)
}

func printOverallSummary(w io.Writer, records []TraceRecord, statsMap map[string]*PerformanceStats) {
	if len(records) == 0 {
		fmt.Fprintf(w, "No trace records to analyze\n")
		return
	}

	var totalDuration int64
	var errorCount int
	startTime, endTime := getTimeRange(records)
	duration := endTime.Sub(startTime)

	for _, record := range records {
		totalDuration += record.DurationMs
		if record.Status == "error" {
			errorCount++
		}
	}

	qps := float64(len(records)) / duration.Seconds()
	avgLatency := float64(totalDuration) / float64(len(records))
	successRate := float64(len(records)-errorCount) / float64(len(records)) * 100

	fmt.Fprintf(w, "┌─ OVERALL PERFORMANCE SUMMARY ─────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│ Test Duration         : %-50s │\n", duration.String())
	fmt.Fprintf(w, "│ Total Requests        : %-50d │\n", len(records))
	fmt.Fprintf(w, "│ Queries Per Second    : %-50.2f │\n", qps)
	fmt.Fprintf(w, "│ Average Latency       : %-50.2f ms │\n", avgLatency)
	fmt.Fprintf(w, "│ Success Rate          : %-50.2f%% │\n", successRate)
	fmt.Fprintf(w, "│ Error Rate            : %-50.2f%% │\n", 100-successRate)
	fmt.Fprintf(w, "│ Unique Operations     : %-50d │\n", len(statsMap))
	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
}

func printTopSlowOperations(w io.Writer, statsMap map[string]*PerformanceStats, topN int) {
	var statsList []*PerformanceStats
	for _, stats := range statsMap {
		statsList = append(statsList, stats)
	}

	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].P99Ms > statsList[j].P99Ms
	})

	fmt.Fprintf(w, "│ %-25s %-15s %8s %8s %8s %8s │\n", "Operation", "Service", "Count", "P95(ms)", "P99(ms)", "Errors%")
	fmt.Fprintf(w, "│ %s │\n", strings.Repeat("─", 74))

	displayCount := topN
	if len(statsList) < topN {
		displayCount = len(statsList)
	}

	for i := 0; i < displayCount; i++ {
		stats := statsList[i]
		fmt.Fprintf(w, "│ %-25s %-15s %8d %8d %8d %7.1f │\n",
			truncateString(stats.Operation, 25),
			truncateString(stats.Service, 15),
			stats.Count,
			stats.P95Ms,
			stats.P99Ms,
			stats.ErrorRate)
	}
}

func printErrorAnalysis(w io.Writer, records []TraceRecord, statsMap map[string]*PerformanceStats) {
	var totalErrors int
	errorsByService := make(map[string]int)
	errorsByOperation := make(map[string]int)

	for _, record := range records {
		if record.Status == "error" {
			totalErrors++
			errorsByService[record.Service]++
			errorsByOperation[record.Operation]++
		}
	}

	if totalErrors == 0 {
		fmt.Fprintf(w, "\n┌─ ERROR ANALYSIS ──────────────────────────────────────────────────────────┐\n")
		fmt.Fprintf(w, "│ ✓ No errors detected during the pressure test                             │\n")
		fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
		return
	}

	fmt.Fprintf(w, "\n┌─ ERROR ANALYSIS ──────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│ Total Errors: %d (%.2f%% of all requests)                                 │\n",
		totalErrors, float64(totalErrors)/float64(len(records))*100)
	fmt.Fprintf(w, "│                                                                            │\n")
	fmt.Fprintf(w, "│ Top Error-Prone Services:                                                  │\n")

	// Sort services by error count
	type serviceError struct {
		name  string
		count int
	}
	var serviceErrors []serviceError
	for service, count := range errorsByService {
		serviceErrors = append(serviceErrors, serviceError{service, count})
	}
	sort.Slice(serviceErrors, func(i, j int) bool {
		return serviceErrors[i].count > serviceErrors[j].count
	})

	for i, se := range serviceErrors {
		if i >= 5 { // Show top 5
			break
		}
		fmt.Fprintf(w, "│   %d. %-30s: %d errors                                 │\n", i+1, se.name, se.count)
	}

	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
}

func printQPSAnalysis(w io.Writer, records []TraceRecord, timeWindow string) {
	if len(records) == 0 {
		return
	}

	windowDuration := parseTimeWindow(timeWindow)
	if windowDuration == 0 {
		windowDuration = 5 * time.Second // Default 5-second windows
	}

	startTime, endTime := getTimeRange(records)
	windows := generateTimeWindows(startTime, endTime, windowDuration)

	fmt.Fprintf(w, "\n┌─ QPS ANALYSIS (%.0fs windows) ────────────────────────────────────────────┐\n", windowDuration.Seconds())

	for i, window := range windows {
		if i >= 10 { // Show first 10 windows
			break
		}
		count := countRequestsInWindow(records, window.StartTime, window.EndTime)
		qps := float64(count) / windowDuration.Seconds()
		fmt.Fprintf(w, "│ %s - %s: %4d req (%.1f QPS) │\n",
			window.StartTime.Format("15:04:05"),
			window.EndTime.Format("15:04:05"),
			count, qps)
	}

	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
}

func printPerformanceTrends(w io.Writer, records []TraceRecord, timeWindow string) {
	if len(records) == 0 {
		return
	}

	windowDuration := parseTimeWindow(timeWindow)
	if windowDuration == 0 {
		windowDuration = 10 * time.Second
	}

	startTime, endTime := getTimeRange(records)
	windows := generateTimeWindows(startTime, endTime, windowDuration)

	fmt.Fprintf(w, "\n┌─ PERFORMANCE TRENDS OVER TIME ────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│ %-10s %8s %10s %10s %8s │\n", "Time", "Requests", "Avg(ms)", "P95(ms)", "Errors")
	fmt.Fprintf(w, "│ %s │\n", strings.Repeat("─", 54))

	for i, window := range windows {
		if i >= 8 { // Show first 8 windows
			break
		}
		windowRecords := getRecordsInWindow(records, window.StartTime, window.EndTime)
		if len(windowRecords) == 0 {
			continue
		}

		avgLatency, p95Latency, errorCount := calculateWindowMetrics(windowRecords)
		fmt.Fprintf(w, "│ %-10s %8d %10.1f %10d %8d │\n",
			window.StartTime.Format("15:04:05"),
			len(windowRecords),
			avgLatency,
			p95Latency,
			errorCount)
	}

	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
}

func printServiceBreakdown(w io.Writer, records []TraceRecord, statsMap map[string]*PerformanceStats) {
	serviceMetrics := make(map[string]ServiceMetrics)

	// Calculate per-service metrics
	for _, stats := range statsMap {
		existing := serviceMetrics[stats.Service]
		existing.RequestCount += stats.Count
		existing.ErrorRate = (existing.ErrorRate*float64(existing.RequestCount-stats.Count) + stats.ErrorRate*float64(stats.Count)) / float64(existing.RequestCount)

		if stats.AvgMs > existing.AvgLatency {
			existing.AvgLatency = stats.AvgMs
		}
		if stats.P99Ms > int64(existing.P99Latency) {
			existing.P99Latency = float64(stats.P99Ms)
		}

		serviceMetrics[stats.Service] = existing
	}

	// Calculate QPS per service
	startTime, endTime := getTimeRange(records)
	duration := endTime.Sub(startTime).Seconds()
	for service := range serviceMetrics {
		metrics := serviceMetrics[service]
		metrics.QPS = float64(metrics.RequestCount) / duration
		serviceMetrics[service] = metrics
	}

	fmt.Fprintf(w, "\n┌─ SERVICE BREAKDOWN ───────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│ %-20s %8s %8s %8s %8s %8s │\n", "Service", "Requests", "QPS", "Avg(ms)", "P99(ms)", "Errors%")
	fmt.Fprintf(w, "│ %s │\n", strings.Repeat("─", 74))

	for service, metrics := range serviceMetrics {
		fmt.Fprintf(w, "│ %-20s %8d %8.1f %8.1f %8.0f %7.1f │\n",
			truncateString(service, 20),
			metrics.RequestCount,
			metrics.QPS,
			metrics.AvgLatency,
			metrics.P99Latency,
			metrics.ErrorRate)
	}

	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
}

func printOptimizationRecommendations(w io.Writer, statsMap map[string]*PerformanceStats, records []TraceRecord) {
	fmt.Fprintf(w, "\n┌─ OPTIMIZATION RECOMMENDATIONS ────────────────────────────────────────────┐\n")

	// Find high-latency operations
	var slowOps []*PerformanceStats
	for _, stats := range statsMap {
		if stats.P95Ms > 500 { // > 500ms is considered slow
			slowOps = append(slowOps, stats)
		}
	}

	if len(slowOps) > 0 {
		fmt.Fprintf(w, "│ 🔥 HIGH LATENCY OPERATIONS DETECTED:                                       │\n")
		for i, stats := range slowOps {
			if i >= 3 {
				break
			}
			fmt.Fprintf(w, "│   • %s.%s: P95=%dms (consider caching/optimization)           │\n",
				stats.Service, stats.Operation, stats.P95Ms)
		}
		fmt.Fprintf(w, "│                                                                            │\n")
	}

	// Find high error rate operations
	var errorOps []*PerformanceStats
	for _, stats := range statsMap {
		if stats.ErrorRate > 1 {
			errorOps = append(errorOps, stats)
		}
	}

	if len(errorOps) > 0 {
		fmt.Fprintf(w, "│ ⚠️  HIGH ERROR RATE OPERATIONS:                                             │\n")
		for i, stats := range errorOps {
			if i >= 3 {
				break
			}
			fmt.Fprintf(w, "│   • %s.%s: %.1f%% error rate                                     │\n",
				stats.Service, stats.Operation, stats.ErrorRate)
		}
		fmt.Fprintf(w, "│                                                                            │\n")
	}

	// General recommendations based on overall performance
	totalRequests := len(records)
	var totalErrors int
	for _, record := range records {
		if record.Status == "error" {
			totalErrors++
		}
	}
	overallErrorRate := float64(totalErrors) / float64(totalRequests) * 100

	if overallErrorRate > 5 {
		fmt.Fprintf(w, "│ 📋 RECOMMENDATIONS:                                                        │\n")
		fmt.Fprintf(w, "│   • Implement circuit breakers to handle cascading failures               │\n")
		fmt.Fprintf(w, "│   • Add retries with exponential backoff                                  │\n")
		fmt.Fprintf(w, "│   • Review service dependencies and timeout configurations                │\n")
	} else if len(slowOps) > 0 {
		fmt.Fprintf(w, "│ 📋 RECOMMENDATIONS:                                                        │\n")
		fmt.Fprintf(w, "│   • Implement caching for frequent operations                             │\n")
		fmt.Fprintf(w, "│   • Consider database query optimization                                  │\n")
		fmt.Fprintf(w, "│   • Review connection pooling configurations                              │\n")
	} else {
		fmt.Fprintf(w, "│ ✅ PERFORMANCE LOOKS GOOD!                                                 │\n")
		fmt.Fprintf(w, "│   • Error rate is acceptable (%.1f%%)                                     │\n", overallErrorRate)
		fmt.Fprintf(w, "│   • Response times are within reasonable ranges                           │\n")
		fmt.Fprintf(w, "│   • Consider stress testing with higher load                              │\n")
	}

	fmt.Fprintf(w, "└────────────────────────────────────────────────────────────────────────────┘\n")
}

// Helper functions for pressure testing analysis

func getTimeRange(records []TraceRecord) (time.Time, time.Time) {
	if len(records) == 0 {
		return time.Time{}, time.Time{}
	}

	var earliest, latest time.Time
	for i, record := range records {
		t, err := time.Parse(time.RFC3339Nano, record.Timestamp)
		if err != nil {
			continue
		}
		if i == 0 {
			earliest, latest = t, t
		} else {
			if t.Before(earliest) {
				earliest = t
			}
			if t.After(latest) {
				latest = t
			}
		}
	}
	return earliest, latest
}

func parseTimeWindow(window string) time.Duration {
	if window == "" {
		return 0
	}
	duration, err := time.ParseDuration(window)
	if err != nil {
		return 0
	}
	return duration
}

func generateTimeWindows(start, end time.Time, windowSize time.Duration) []TimeWindowStats {
	var windows []TimeWindowStats
	current := start

	for current.Before(end) {
		windowEnd := current.Add(windowSize)
		if windowEnd.After(end) {
			windowEnd = end
		}
		windows = append(windows, TimeWindowStats{
			StartTime: current,
			EndTime:   windowEnd,
		})
		current = windowEnd
	}

	return windows
}

func countRequestsInWindow(records []TraceRecord, start, end time.Time) int {
	count := 0
	for _, record := range records {
		t, err := time.Parse(time.RFC3339Nano, record.Timestamp)
		if err != nil {
			continue
		}
		if t.After(start) && t.Before(end) {
			count++
		}
	}
	return count
}

func getRecordsInWindow(records []TraceRecord, start, end time.Time) []TraceRecord {
	var windowRecords []TraceRecord
	for _, record := range records {
		t, err := time.Parse(time.RFC3339Nano, record.Timestamp)
		if err != nil {
			continue
		}
		if t.After(start) && t.Before(end) {
			windowRecords = append(windowRecords, record)
		}
	}
	return windowRecords
}

func calculateWindowMetrics(records []TraceRecord) (float64, int64, int) {
	if len(records) == 0 {
		return 0, 0, 0
	}

	var totalDuration int64
	var durations []int64
	var errorCount int

	for _, record := range records {
		totalDuration += record.DurationMs
		durations = append(durations, record.DurationMs)
		if record.Status == "error" {
			errorCount++
		}
	}

	avgLatency := float64(totalDuration) / float64(len(records))

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	p95Latency := percentile(durations, 95)

	return avgLatency, p95Latency, errorCount
}
