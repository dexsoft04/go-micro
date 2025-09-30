package metadata

import (
	"strings"

	"go-micro.dev/v5/transport/headers"
)

// Framework internal control headers that should not be propagated between services.
// These headers are set by the framework for each RPC call and describe the current call,
// not the original call chain.
var frameworkControlHeaders = map[string]bool{
	"micro-service":  true, // Target service name for current call
	"micro-method":   true, // Target method name for current call
	"micro-endpoint": true, // Target endpoint for current call
	"micro-id":       true, // Request ID for current call
	"micro-protocol": true, // Protocol negotiation for current call
	"micro-target":   true, // Target address for current call
}

// HTTP hop-by-hop headers that should not be forwarded between services.
// These are connection-specific and must not be propagated.
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

// Server-local headers that contain connection-specific information.
// These should not be forwarded to downstream services.
var serverLocalHeaders = map[string]bool{
	"local":  true, // Local endpoint address
	"remote": true, // Remote client address
}

// ShouldFilterFrameworkHeader checks if a header is a framework control header
// that should be filtered out when forwarding to downstream services.
// Returns true if the header should be filtered (not propagated).
func ShouldFilterFrameworkHeader(key string) bool {
	keyLower := strings.ToLower(key)

	// Filter hop-by-hop headers (HTTP standard)
	if hopByHopHeaders[keyLower] {
		return true
	}

	// Filter server-local headers (connection-specific)
	if serverLocalHeaders[keyLower] {
		return true
	}

	// Filter framework control headers (RPC call-specific)
	if frameworkControlHeaders[keyLower] {
		return true
	}

	// Filter Micro-Topic header (pub/sub specific, should not leak to RPC calls)
	if keyLower == strings.ToLower(headers.Message) {
		return true
	}

	// Don't filter anything else - let application layer decide
	return false
}

// FilterFrameworkHeaders removes framework control headers from metadata.
// This is the minimal filtering that the framework layer should perform.
// Application-level filtering (auth, tracing, business headers) should be done
// at the API gateway/entry point, not in the framework.
func FilterFrameworkHeaders(md Metadata) Metadata {
	if md == nil {
		return nil
	}

	filtered := make(Metadata)
	for key, value := range md {
		if !ShouldFilterFrameworkHeader(key) {
			filtered[key] = value
		}
	}

	return filtered
}

// FilterIncomingHeaders filters incoming metadata to remove headers that should
// not be stored in the service context (e.g., hop-by-hop headers).
// This keeps only transport-layer filtering and preserves all application headers.
func FilterIncomingHeaders(md Metadata) Metadata {
	if md == nil {
		return nil
	}

	filtered := make(Metadata)
	for key, value := range md {
		keyLower := strings.ToLower(key)

		// Skip only hop-by-hop headers (transport layer)
		if hopByHopHeaders[keyLower] {
			continue
		}

		// Include all other headers for local processing
		filtered[key] = value
	}

	return filtered
}
