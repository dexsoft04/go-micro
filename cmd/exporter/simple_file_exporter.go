package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go-micro.dev/v5/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// SimpleFileExporter exports traces to a file in simplified JSON format
type SimpleFileExporter struct {
	file     *os.File
	mu       sync.Mutex
	encoder  *json.Encoder
	filePath string
}

// TraceRecord represents a simplified trace record for performance analysis
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

// NewSimpleFileExporter creates a new file exporter with simplified JSON output
func NewSimpleFileExporter(filepath string) (*SimpleFileExporter, error) {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filepath, err)
	}

	exporter := &SimpleFileExporter{
		file:     file,
		encoder:  json.NewEncoder(file),
		filePath: filepath,
	}

	logger.Infof("Created simple file exporter: %s", filepath)
	return exporter, nil
}

// ExportSpans exports the given spans to the file
func (e *SimpleFileExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, span := range spans {
		record := e.convertSpanToRecord(span)
		if err := e.encoder.Encode(record); err != nil {
			logger.Errorf("failed to write trace record: %v", err)
			return err
		}
	}

	// Flush to ensure data is written
	if err := e.file.Sync(); err != nil {
		logger.Warnf("failed to sync trace file: %v", err)
	}

	return nil
}

// convertSpanToRecord converts an OpenTelemetry span to a simplified trace record
func (e *SimpleFileExporter) convertSpanToRecord(span trace.ReadOnlySpan) TraceRecord {
	// Extract service name from resource attributes
	serviceName := "unknown"
	if serviceAttr, exists := span.Resource().Set().Value(semconv.ServiceNameKey); exists && serviceAttr.Type() == attribute.STRING {
		serviceName = serviceAttr.AsString()
	}

	// Calculate duration
	duration := span.EndTime().Sub(span.StartTime())

	// Determine status
	status := "ok"
	errorMsg := ""
	if span.Status().Code != 0 { // 0 = Unset, 1 = Error, 2 = Ok
		if span.Status().Code == 1 {
			status = "error"
			errorMsg = span.Status().Description
		}
	}

	// Convert attributes to map
	attrs := make(map[string]interface{})
	for _, attr := range span.Attributes() {
		key := string(attr.Key)
		switch attr.Value.Type() {
		case attribute.STRING:
			attrs[key] = attr.Value.AsString()
		case attribute.INT64:
			attrs[key] = attr.Value.AsInt64()
		case attribute.FLOAT64:
			attrs[key] = attr.Value.AsFloat64()
		case attribute.BOOL:
			attrs[key] = attr.Value.AsBool()
		default:
			attrs[key] = attr.Value.AsString()
		}
	}

	// Get parent span ID if exists
	parentSpanID := ""
	if span.Parent().IsValid() {
		parentSpanID = span.Parent().SpanID().String()
	}

	record := TraceRecord{
		Timestamp:    time.Now().Format(time.RFC3339),
		Service:      serviceName,
		Operation:    span.Name(),
		DurationMs:   duration.Milliseconds(),
		TraceID:      span.SpanContext().TraceID().String(),
		SpanID:       span.SpanContext().SpanID().String(),
		ParentSpanID: parentSpanID,
		Status:       status,
		Attributes:   attrs,
		Error:        errorMsg,
	}

	return record
}

// Shutdown closes the file exporter
func (e *SimpleFileExporter) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.file != nil {
		err := e.file.Close()
		logger.Infof("Closed simple file exporter: %s", e.filePath)
		return err
	}
	return nil
}

// ForceFlush forces any buffered data to be written
func (e *SimpleFileExporter) ForceFlush(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.file != nil {
		return e.file.Sync()
	}
	return nil
}