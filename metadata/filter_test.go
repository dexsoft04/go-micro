package metadata

import (
	"testing"

	"go-micro.dev/v5/transport/headers"
)

func TestShouldPropagate(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		// Hop-by-hop headers should not propagate
		{"hop-by-hop Connection", "Connection", false},
		{"hop-by-hop Keep-Alive", "Keep-Alive", false},
		{"hop-by-hop case insensitive", "connection", false},
		{"hop-by-hop Transfer-Encoding", "Transfer-Encoding", false},

		// Server-local headers should not propagate
		{"server-local Local", "Local", false},
		{"server-local Remote", "Remote", false},
		{"server-local case insensitive", "local", false},

		// Framework control headers should not propagate
		{"framework Micro-Service", "Micro-Service", false},
		{"framework Micro-Method", "Micro-Method", false},
		{"framework case insensitive", "micro-service", false},

		// Micro-Topic should not propagate (pub/sub)
		{headers.Message, headers.Message, false},

		// Tracing headers should propagate
		{"tracing X-Request-ID", "X-Request-ID", true},
		{"tracing X-Trace-ID", "X-Trace-ID", true},
		{"tracing case insensitive", "x-request-id", true},
		{"tracing X-B3-TraceId", "X-B3-TraceId", true},
		{"tracing Uber-Trace-Id", "Uber-Trace-Id", true},

		// Auth headers should propagate
		{"auth Authorization", "Authorization", true},
		{"auth X-API-Key", "X-API-Key", true},
		{"auth case insensitive", "authorization", true},
		{"auth User-Token", "User-Token", true},
		{"auth User-Authorization", "User-Authorization", true},
		{"auth igs-user-id", "igs-user-id", true},

		// Content headers should propagate
		{"content Content-Type", "Content-Type", true},
		{"content Accept", "Accept", true},

		// Custom business headers should propagate
		{"business custom header", "X-Custom-Business", true},
		{"business User-ID", "User-ID", true},

		// Micro namespace and trace headers should propagate
		{"micro namespace", headers.Namespace, true},
		{"micro span id", headers.SpanID, true},
		{"micro trace id", headers.TraceIDKey, true},

		// WebSocket session headers should propagate
		{"websocket micro-ws-session-id", "micro-ws-session-id", true},
		{"websocket micro-ws-server-id", "micro-ws-server-id", true},
		{"websocket case insensitive", "Micro-WS-Session-ID", true},

		// Unknown Micro- headers should not propagate
		{"unknown micro header", "Micro-Unknown", false},

		// X-Forwarded headers should not propagate
		{"forwarded X-Forwarded-For", "X-Forwarded-For", false},
		{"forwarded X-Real-IP", "X-Real-IP", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldPropagate(tt.key)
			if result != tt.expected {
				t.Errorf("ShouldPropagate(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestFilterForwardHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    Metadata
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    Metadata{},
			expected: map[string]string{},
		},
		{
			name: "mixed headers",
			input: Metadata{
				"Authorization":   "Bearer token123",
				"X-Request-ID":    "req-123",
				"Connection":      "keep-alive",
				"Local":           "127.0.0.1:8080",
				"Micro-Service":   "test.service",
				"Micro-Topic":     "test.topic",
				"Content-Type":    "application/json",
				"X-Custom-Header": "custom-value",
				"Micro-Namespace": "production",
				"X-Forwarded-For": "10.0.0.1",
			},
			expected: map[string]string{
				"Authorization":   "Bearer token123",
				"X-Request-ID":    "req-123",
				"Content-Type":    "application/json",
				"X-Custom-Header": "custom-value",
				"Micro-Namespace": "production",
			},
		},
		{
			name: "tracing headers",
			input: Metadata{
				"X-Trace-ID":      "trace-123",
				"X-B3-TraceId":    "b3-trace-123",
				"X-B3-SpanId":     "b3-span-123",
				"Uber-Trace-Id":   "uber-trace-123",
				"Jaeger-Debug-Id": "jaeger-debug-123",
				"Connection":      "close",
			},
			expected: map[string]string{
				"X-Trace-ID":      "trace-123",
				"X-B3-TraceId":    "b3-trace-123",
				"X-B3-SpanId":     "b3-span-123",
				"Uber-Trace-Id":   "uber-trace-123",
				"Jaeger-Debug-Id": "jaeger-debug-123",
			},
		},
		{
			name: "websocket headers",
			input: Metadata{
				"Authorization":       "Bearer token123",
				"micro-ws-session-id": "session-abc-123",
				"micro-ws-server-id":  "ws-server-01",
				"Connection":          "upgrade",
				"Micro-Service":       "websocket.service",
				"X-Request-ID":        "req-456",
			},
			expected: map[string]string{
				"Authorization":       "Bearer token123",
				"micro-ws-session-id": "session-abc-123",
				"micro-ws-server-id":  "ws-server-01",
				"X-Request-ID":        "req-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterForwardHeaders(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("FilterForwardHeaders() = %v, want nil", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("FilterForwardHeaders() length = %d, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("FilterForwardHeaders() missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("FilterForwardHeaders()[%q] = %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("FilterForwardHeaders() unexpected key %q with value %q", key, result[key])
				}
			}
		})
	}
}

func TestFilterIncomingHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    Metadata
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "filter hop-by-hop headers",
			input: Metadata{
				"Authorization": "Bearer token123",
				"Connection":    "keep-alive",
				"Keep-Alive":    "timeout=5",
				"Content-Type":  "application/json",
			},
			expected: map[string]string{
				"Authorization": "Bearer token123",
				"Content-Type":  "application/json",
			},
		},
		{
			name: "preserve all non-hop-by-hop headers",
			input: Metadata{
				"Local":             "127.0.0.1:8080",
				"Remote":            "192.168.1.100:12345",
				"Micro-Service":     "test.service",
				"X-Request-ID":      "req-123",
				"Transfer-Encoding": "chunked",
			},
			expected: map[string]string{
				"Local":         "127.0.0.1:8080",
				"Remote":        "192.168.1.100:12345",
				"Micro-Service": "test.service",
				"X-Request-ID":  "req-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterIncomingHeaders(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("FilterIncomingHeaders() = %v, want nil", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("FilterIncomingHeaders() length = %d, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("FilterIncomingHeaders() missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("FilterIncomingHeaders()[%q] = %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("FilterIncomingHeaders() unexpected key %q with value %q", key, result[key])
				}
			}
		})
	}
}

func TestCallChainScenario(t *testing.T) {
	// Simulate A -> B -> C call chain
	// Service A sends request to B, B forwards some headers to C

	// Original headers from service A
	originalHeaders := Metadata{
		"Authorization":     "Bearer token123",
		"X-Request-ID":      "req-abc-123",
		"Content-Type":      "application/json",
		"User-Agent":        "service-a/1.0",
		"X-Custom-Business": "business-value",
	}

	// Service B receives and processes these headers
	incomingFiltered := FilterIncomingHeaders(originalHeaders)

	// Service B adds local information
	serviceB_Context := Copy(incomingFiltered)
	serviceB_Context["Local"] = "127.0.0.1:8081"
	serviceB_Context["Remote"] = "127.0.0.1:8080"
	serviceB_Context["Micro-Service"] = "service.b"

	// Service B makes call to C, filtering headers for forward propagation
	forwardToC := FilterForwardHeaders(serviceB_Context)

	// Verify the call chain behavior
	expectedForwardToC := map[string]string{
		"Authorization":     "Bearer token123",
		"X-Request-ID":      "req-abc-123",
		"Content-Type":      "application/json",
		"User-Agent":        "service-a/1.0",
		"X-Custom-Business": "business-value",
	}

	if len(forwardToC) != len(expectedForwardToC) {
		t.Errorf("Forward to C length = %d, want %d", len(forwardToC), len(expectedForwardToC))
		t.Errorf("Actual forwardToC: %+v", forwardToC)
		t.Errorf("Expected: %+v", expectedForwardToC)
	}

	for key, expectedValue := range expectedForwardToC {
		if actualValue, exists := forwardToC[key]; !exists {
			t.Errorf("Forward to C missing key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("Forward to C[%q] = %q, want %q", key, actualValue, expectedValue)
		}
	}

	// Verify that Local, Remote, and Micro-Service are not forwarded
	forbiddenKeys := []string{"Local", "Remote", "Micro-Service"}
	for _, key := range forbiddenKeys {
		if _, exists := forwardToC[key]; exists {
			t.Errorf("Forward to C should not contain key %q, but it does with value %q", key, forwardToC[key])
		}
	}
}
