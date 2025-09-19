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
	Operation    string
	Service      string
	Count        int
	TotalMs      int64
	MinMs        int64
	MaxMs        int64
	AvgMs        float64
	P50Ms        int64
	P95Ms        int64
	P99Ms        int64
	ErrorCount   int
	ErrorRate    float64
	Durations    []int64
}

func main() {
	var (
		inputFile     = flag.String("input", "", "Input trace file (JSONL format)")
		outputFormat  = flag.String("format", "table", "Output format: table, json, html")
		outputFile    = flag.String("output", "", "Output file (default: stdout)")
		serviceFilter = flag.String("service", "", "Filter by service name")
		minDuration   = flag.Int64("min-duration", 0, "Filter by minimum duration (ms)")
		showErrors    = flag.Bool("errors", false, "Show only errors")
		streamMode    = flag.Bool("stream", false, "Stream mode: analyze traces in real-time")
		topN          = flag.Int("top", 10, "Show top N slowest operations")
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