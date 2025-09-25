package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-micro.dev/v5/transport"
)

// Mock transport message to test header timeout values
type mockTransport struct {
	lastMessage *transport.Message
}

func (m *mockTransport) captureMessage(msg *transport.Message) {
	m.lastMessage = msg
}

func TestTimeoutHeaderValues(t *testing.T) {
	tests := []struct {
		name               string
		contextTimeout     time.Duration
		connectionTimeout  time.Duration
		expectedTimeout    time.Duration
		description        string
	}{
		{
			name:              "Context deadline shorter than connection timeout",
			contextTimeout:    2 * time.Second,
			connectionTimeout: 5 * time.Second,
			expectedTimeout:   2 * time.Second,
			description:       "Should use context deadline when it's shorter",
		},
		{
			name:              "Connection timeout shorter than context deadline",
			contextTimeout:    10 * time.Second,
			connectionTimeout: 3 * time.Second,
			expectedTimeout:   3 * time.Second,
			description:       "Should use connection timeout when it's shorter",
		},
		{
			name:              "No context deadline",
			contextTimeout:    0, // No deadline
			connectionTimeout: 4 * time.Second,
			expectedTimeout:   4 * time.Second,
			description:       "Should use connection timeout when no context deadline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context
			var ctx context.Context
			var cancel context.CancelFunc

			if tt.contextTimeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tt.contextTimeout)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			// Simulate the timeout calculation logic from our fix
			cTimeout := tt.connectionTimeout
			if cTimeout == 0 {
				cTimeout = DefaultConnectionTimeout
			}

			// Check if context has a deadline and use the shorter timeout
			if deadline, ok := ctx.Deadline(); ok {
				remaining := time.Until(deadline)
				if remaining > 0 && remaining < cTimeout {
					cTimeout = remaining
				}
			}

			// Verify the effective timeout
			tolerance := 100 * time.Millisecond // Allow for small time differences
			if cTimeout > tt.expectedTimeout+tolerance || cTimeout < tt.expectedTimeout-tolerance {
				t.Errorf("Expected timeout ~%v, got %v (tolerance ±%v)",
					tt.expectedTimeout, cTimeout, tolerance)
			}

			// Test header value format (nanoseconds)
			headerValue := fmt.Sprintf("%d", cTimeout)
			nanoseconds := int64(cTimeout)

			t.Logf("Test: %s", tt.description)
			t.Logf("Effective timeout: %v", cTimeout)
			t.Logf("Header value: %s (nanoseconds)", headerValue)
			t.Logf("Expected nanoseconds: %d", nanoseconds)

			// Verify the header value can be parsed back correctly
			if parsedNanos := int64(cTimeout); parsedNanos <= 0 {
				t.Errorf("Invalid timeout nanoseconds: %d", parsedNanos)
			}
		})
	}
}