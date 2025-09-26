package metadata

import (
	"strings"

	"go-micro.dev/v5/transport/headers"
)

// HTTP hop-by-hop headers that should not be forwarded
var hopByHopHeaders = map[string]bool{
	"connection":          true,
	"proxy-connection":    true,
	"keep-alive":          true,
	"transfer-encoding":   true,
	"upgrade":             true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailer":             true,
}

// Server-local headers that should not be forwarded to downstream services
var serverLocalHeaders = map[string]bool{
	"local":  true,
	"remote": true,
}

// Framework internal control headers that should not be propagated
var frameworkControlHeaders = map[string]bool{
	"micro-service":  true,
	"micro-method":   true,
	"micro-endpoint": true,
	"micro-id":       true,
	"micro-protocol": true,
	"micro-target":   true,
}

// Headers that should be preserved for tracing and monitoring
var tracingHeaders = map[string]bool{
	"x-request-id":      true,
	"x-correlation-id":  true,
	"x-trace-id":        true,
	"x-span-id":         true,
	"x-b3-traceid":      true,
	"x-b3-spanid":       true,
	"x-b3-parentspanid": true,
	"x-b3-sampled":      true,
	"x-b3-flags":        true,
	"uber-trace-id":     true,
	"jaeger-debug-id":   true,
	"jaeger-baggage":    true,
	// W3C Trace Context
	"traceparent": true,
	"tracestate":  true,
	"baggage":     true,
}

// WebSocket session headers that should be preserved for WebSocket-based services
var websocketHeaders = map[string]bool{
	"micro-ws-session-id": true,
	"micro-ws-server-id":  true,
}

// Authentication and authorization headers that should be preserved
var authHeaders = map[string]bool{
	"authorization":      true,
	"x-api-key":          true,
	"x-auth-token":       true,
	"user-token":         true,
	"user-authorization": true,
	"igs-user-id":        true,
}

// ShouldPropagate checks if a header should be propagated to downstream services
func ShouldPropagate(key string) bool {
	keyLower := strings.ToLower(key)

	// Always filter out hop-by-hop headers
	if hopByHopHeaders[keyLower] {
		return false
	}

	// Always filter out server-local headers
	if serverLocalHeaders[keyLower] {
		return false
	}

	// Always filter out framework control headers
	if frameworkControlHeaders[keyLower] {
		return false
	}

	// Skip Micro-Topic header used for pub/sub
	if keyLower == strings.ToLower(headers.Message) {
		return false
	}

	// Always preserve tracing headers
	if tracingHeaders[keyLower] {
		return true
	}

	// Always preserve WebSocket session headers
	if websocketHeaders[keyLower] {
		return true
	}

	// Always preserve auth headers
	if authHeaders[keyLower] {
		return true
	}

	// Preserve content-type and accept headers
	if keyLower == "content-type" || keyLower == "accept" {
		return true
	}

	// Preserve custom business headers (not starting with known system prefixes)
	if !strings.HasPrefix(keyLower, "micro-") &&
		!strings.HasPrefix(keyLower, "x-forwarded-") &&
		!strings.HasPrefix(keyLower, "x-real-") {
		return true
	}

	// Preserve Micro-Namespace if present (may be needed for routing)
	if keyLower == strings.ToLower(headers.Namespace) {
		return true
	}

	// Preserve trace-related Micro headers
	if keyLower == strings.ToLower(headers.SpanID) || keyLower == strings.ToLower(headers.TraceIDKey) {
		return true
	}

	// Default: don't propagate unknown Micro- prefixed headers
	return false
}

// FilterForwardHeaders filters metadata to include only headers that should be
// propagated to downstream services in a call chain
func FilterForwardHeaders(md Metadata) Metadata {
	if md == nil {
		return nil
	}

	filtered := make(Metadata)
	for key, value := range md {
		if ShouldPropagate(key) {
			filtered[key] = value
		}
	}

	return filtered
}

// FilterIncomingHeaders filters incoming metadata to remove headers that should
// not be stored in the service context (e.g., server-generated headers)
func FilterIncomingHeaders(md Metadata) Metadata {
	if md == nil {
		return nil
	}

	filtered := make(Metadata)
	for key, value := range md {
		keyLower := strings.ToLower(key)

		// Skip headers that are only for transport layer
		if hopByHopHeaders[keyLower] {
			continue
		}

		// Include all other headers for local processing
		filtered[key] = value
	}

	return filtered
}
