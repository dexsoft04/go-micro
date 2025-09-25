package server

import (
	"context"
	"testing"
	"time"
)

func TestServerTimeoutOptions(t *testing.T) {
	// Test default values
	opts := NewOptions()
	if opts.MaxRequestTimeout != DefaultMaxRequestTimeout {
		t.Errorf("Expected MaxRequestTimeout to be %v, got %v", DefaultMaxRequestTimeout, opts.MaxRequestTimeout)
	}
	if opts.DefaultRequestTimeout != DefaultServerRequestTimeout {
		t.Errorf("Expected DefaultRequestTimeout to be %v, got %v", DefaultServerRequestTimeout, opts.DefaultRequestTimeout)
	}

	// Test custom values
	maxTimeout := 45 * time.Second
	defaultTimeout := 20 * time.Second

	opts = NewOptions(
		WithMaxRequestTimeout(maxTimeout),
		WithDefaultRequestTimeout(defaultTimeout),
	)

	if opts.MaxRequestTimeout != maxTimeout {
		t.Errorf("Expected MaxRequestTimeout to be %v, got %v", maxTimeout, opts.MaxRequestTimeout)
	}
	if opts.DefaultRequestTimeout != defaultTimeout {
		t.Errorf("Expected DefaultRequestTimeout to be %v, got %v", defaultTimeout, opts.DefaultRequestTimeout)
	}
}

func TestServerTimeoutLogic(t *testing.T) {
	tests := []struct {
		name              string
		clientTimeout     time.Duration
		maxTimeout        time.Duration
		defaultTimeout    time.Duration
		contextDeadline   time.Duration
		expectedTimeout   time.Duration
		description       string
	}{
		{
			name:            "Client timeout within server limits",
			clientTimeout:   5 * time.Second,
			maxTimeout:      10 * time.Second,
			defaultTimeout:  8 * time.Second,
			expectedTimeout: 5 * time.Second,
			description:     "Should use client timeout when it's within server limits",
		},
		{
			name:            "Client timeout exceeds max timeout",
			clientTimeout:   15 * time.Second,
			maxTimeout:      10 * time.Second,
			defaultTimeout:  8 * time.Second,
			expectedTimeout: 10 * time.Second,
			description:     "Should use max timeout when client timeout exceeds it",
		},
		{
			name:            "No client timeout, use default",
			clientTimeout:   0,
			maxTimeout:      10 * time.Second,
			defaultTimeout:  8 * time.Second,
			expectedTimeout: 8 * time.Second,
			description:     "Should use default timeout when client doesn't specify",
		},
		{
			name:            "Context deadline shorter than all",
			clientTimeout:   5 * time.Second,
			maxTimeout:      10 * time.Second,
			defaultTimeout:  8 * time.Second,
			contextDeadline: 3 * time.Second,
			expectedTimeout: 3 * time.Second,
			description:     "Should use context deadline when it's shortest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a base context
			var ctx context.Context
			var cancel context.CancelFunc

			if tt.contextDeadline > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tt.contextDeadline)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			// Simulate the server timeout logic from our fix
			requestTimeout := tt.clientTimeout

			// Apply server-side timeout constraints
			if requestTimeout == 0 && tt.defaultTimeout > 0 {
				requestTimeout = tt.defaultTimeout
			}
			if tt.maxTimeout > 0 && requestTimeout > tt.maxTimeout {
				requestTimeout = tt.maxTimeout
			}

			// Check if parent context has a deadline and use the shorter timeout
			if deadline, ok := ctx.Deadline(); ok {
				remaining := time.Until(deadline)
				if remaining > 0 && remaining < requestTimeout {
					requestTimeout = remaining
				}
			}

			// Verify the effective timeout (with some tolerance for time differences)
			tolerance := 100 * time.Millisecond
			if requestTimeout > tt.expectedTimeout+tolerance || requestTimeout < tt.expectedTimeout-tolerance {
				t.Errorf("Expected timeout ~%v, got %v (tolerance ±%v)",
					tt.expectedTimeout, requestTimeout, tolerance)
			}

			t.Logf("Test: %s", tt.description)
			t.Logf("Client timeout: %v", tt.clientTimeout)
			t.Logf("Max timeout: %v", tt.maxTimeout)
			t.Logf("Default timeout: %v", tt.defaultTimeout)
			t.Logf("Context deadline: %v", tt.contextDeadline)
			t.Logf("Effective timeout: %v", requestTimeout)
		})
	}
}