package metadata

import (
	"testing"

	"go-micro.dev/v5/transport/headers"
)

func TestShouldFilterFrameworkHeader(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		shouldFilter bool // true if header should be filtered (not propagated)
	}{
		// Hop-by-hop headers should be filtered
		{"hop-by-hop Connection", "Connection", true},
		{"hop-by-hop Keep-Alive", "Keep-Alive", true},
		{"hop-by-hop case insensitive", "connection", true},
		{"hop-by-hop Transfer-Encoding", "Transfer-Encoding", true},

		// Server-local headers should be filtered
		{"server-local Local", "Local", true},
		{"server-local Remote", "Remote", true},
		{"server-local case insensitive", "local", true},

		// Framework control headers should be filtered
		{"framework Micro-Service", "Micro-Service", true},
		{"framework Micro-Method", "Micro-Method", true},
		{"framework case insensitive", "micro-service", true},
		{"framework Micro-ID", "Micro-ID", true},
		{"framework Micro-Protocol", "Micro-Protocol", true},
		{"framework Micro-Target", "Micro-Target", true},

		// Micro-Topic should be filtered (pub/sub)
		{headers.Message, headers.Message, true},

		// Application headers should NOT be filtered
		{"tracing X-Request-ID", "X-Request-ID", false},
		{"tracing X-Trace-ID", "X-Trace-ID", false},
		{"auth Authorization", "Authorization", false},
		{"auth X-API-Key", "X-API-Key", false},
		{"auth User-Token", "User-Token", false},
		{"content Content-Type", "Content-Type", false},
		{"content Accept", "Accept", false},
		{"business X-Custom-Business", "X-Custom-Business", false},
		{"business User-ID", "User-ID", false},

		// Client info headers should NOT be filtered (needed for tracing original client)
		{"client X-Forwarded-For", "X-Forwarded-For", false},
		{"client X-Real-IP", "X-Real-IP", false},
		{"client X-Forwarded-Host", "X-Forwarded-Host", false},

		// Special Micro headers that should NOT be filtered
		{"micro namespace", headers.Namespace, false},
		{"micro span id", headers.SpanID, false},
		{"micro trace id", headers.TraceIDKey, false},

		// WebSocket session headers should NOT be filtered
		{"websocket micro-ws-session-id", "micro-ws-session-id", false},
		{"websocket micro-ws-server-id", "micro-ws-server-id", false},

		// Unknown Micro- headers should NOT be filtered (let application decide)
		{"unknown micro header", "Micro-Unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldFilterFrameworkHeader(tt.key)
			if result != tt.shouldFilter {
				t.Errorf("ShouldFilterFrameworkHeader(%q) = %v, want %v", tt.key, result, tt.shouldFilter)
			}
		})
	}
}

func TestFilterFrameworkHeaders(t *testing.T) {
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
				"Authorization":     "Bearer token123",
				"X-Request-ID":      "req-123",
				"Connection":        "keep-alive",
				"Local":             "127.0.0.1:8080",
				"Micro-Service":     "test.service",
				"Micro-Topic":       "test.topic",
				"Content-Type":      "application/json",
				"X-Custom-Header":   "custom-value",
				"Micro-Namespace":   "production",
				"X-Forwarded-For":   "10.0.0.1",
				"User-Agent":        "test-client/1.0",
				"micro-ws-session-id": "session-123",
			},
			expected: map[string]string{
				"Authorization":       "Bearer token123",
				"X-Request-ID":        "req-123",
				"Content-Type":        "application/json",
				"X-Custom-Header":     "custom-value",
				"Micro-Namespace":     "production",
				"X-Forwarded-For":     "10.0.0.1",
				"User-Agent":          "test-client/1.0",
				"micro-ws-session-id": "session-123",
			},
		},
		{
			name: "only framework headers",
			input: Metadata{
				"Connection":      "close",
				"Micro-Service":   "test.service",
				"Micro-Method":    "TestMethod",
				"Micro-Topic":     "test.topic",
				"Local":           "127.0.0.1:8080",
				"Remote":          "192.168.1.1:12345",
			},
			expected: map[string]string{},
		},
		{
			name: "client info and tracing headers",
			input: Metadata{
				"X-Forwarded-For":   "203.0.113.1",
				"X-Real-IP":         "203.0.113.1",
				"X-Forwarded-Host":  "example.com",
				"X-Forwarded-Proto": "https",
				"X-Trace-ID":        "trace-123",
				"Authorization":     "Bearer token456",
				"Micro-Service":     "gateway.service",
			},
			expected: map[string]string{
				"X-Forwarded-For":   "203.0.113.1",
				"X-Real-IP":         "203.0.113.1",
				"X-Forwarded-Host":  "example.com",
				"X-Forwarded-Proto": "https",
				"X-Trace-ID":        "trace-123",
				"Authorization":     "Bearer token456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterFrameworkHeaders(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("FilterFrameworkHeaders() = %v, want nil", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("FilterFrameworkHeaders() length = %d, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %+v", result)
				t.Errorf("Expected: %+v", tt.expected)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("FilterFrameworkHeaders() missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("FilterFrameworkHeaders()[%q] = %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("FilterFrameworkHeaders() unexpected key %q with value %q", key, result[key])
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
			name: "preserve all non-hop-by-hop headers including framework headers",
			input: Metadata{
				"Local":             "127.0.0.1:8080",
				"Remote":            "192.168.1.100:12345",
				"Micro-Service":     "test.service",
				"X-Request-ID":      "req-123",
				"Transfer-Encoding": "chunked",
				"X-Forwarded-For":   "10.0.0.1",
			},
			expected: map[string]string{
				"Local":           "127.0.0.1:8080",
				"Remote":          "192.168.1.100:12345",
				"Micro-Service":   "test.service",
				"X-Request-ID":    "req-123",
				"X-Forwarded-For": "10.0.0.1",
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
	// Service A (API Gateway) sends request to B, B forwards headers to C

	// Original headers from external client through API gateway
	originalHeaders := Metadata{
		"Authorization":     "Bearer token123",
		"X-Request-ID":      "req-abc-123",
		"X-Forwarded-For":   "203.0.113.1",
		"X-Real-IP":         "203.0.113.1",
		"Content-Type":      "application/json",
		"User-Agent":        "Mozilla/5.0",
		"X-Custom-Business": "business-value",
	}

	// Service B receives these headers
	incomingFiltered := FilterIncomingHeaders(originalHeaders)

	// Service B adds local information (added by framework)
	serviceB_Context := Copy(incomingFiltered)
	serviceB_Context["Local"] = "127.0.0.1:8081"
	serviceB_Context["Remote"] = "127.0.0.1:8080"
	serviceB_Context["Micro-Service"] = "service.b"
	serviceB_Context["Micro-Method"] = "Handler"

	// Service B makes call to C, filtering framework headers
	forwardToC := FilterFrameworkHeaders(serviceB_Context)

	// Verify: all application headers should be forwarded, framework headers filtered
	expectedForwardToC := map[string]string{
		"Authorization":     "Bearer token123",
		"X-Request-ID":      "req-abc-123",
		"X-Forwarded-For":   "203.0.113.1",
		"X-Real-IP":         "203.0.113.1",
		"Content-Type":      "application/json",
		"User-Agent":        "Mozilla/5.0",
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

	// Verify that framework headers are filtered
	forbiddenKeys := []string{"Local", "Remote", "Micro-Service", "Micro-Method"}
	for _, key := range forbiddenKeys {
		if _, exists := forwardToC[key]; exists {
			t.Errorf("Forward to C should not contain key %q, but it does with value %q", key, forwardToC[key])
		}
	}

	// Verify that client info headers are preserved throughout the chain
	clientInfoKeys := []string{"X-Forwarded-For", "X-Real-IP"}
	for _, key := range clientInfoKeys {
		if _, exists := forwardToC[key]; !exists {
			t.Errorf("Forward to C must contain client info key %q for tracing original client", key)
		}
	}
}