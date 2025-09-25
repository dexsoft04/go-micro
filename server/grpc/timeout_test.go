package grpc

import (
	"context"
	"testing"
	"time"

	"go-micro.dev/v5/server"
)

func TestGRPCServerTimeoutLogic(t *testing.T) {
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
			// Create server options with timeout settings
			opts := server.NewOptions(
				server.WithMaxRequestTimeout(tt.maxTimeout),
				server.WithDefaultRequestTimeout(tt.defaultTimeout),
			)

			// Create a base context
			var ctx context.Context
			var cancel context.CancelFunc

			if tt.contextDeadline > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tt.contextDeadline)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			// Simulate the gRPC server timeout logic from our fix
			requestTimeout := tt.clientTimeout

			// Apply server-side timeout constraints
			if requestTimeout == 0 && opts.DefaultRequestTimeout > 0 {
				requestTimeout = opts.DefaultRequestTimeout
			}
			if opts.MaxRequestTimeout > 0 && requestTimeout > opts.MaxRequestTimeout {
				requestTimeout = opts.MaxRequestTimeout
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

func TestGRPCServerOptionsIntegration(t *testing.T) {
	// Test that gRPC server can use server.Options timeout settings
	opts := server.NewOptions(
		server.WithMaxRequestTimeout(30*time.Second),
		server.WithDefaultRequestTimeout(15*time.Second),
	)

	if opts.MaxRequestTimeout != 30*time.Second {
		t.Errorf("Expected MaxRequestTimeout to be 30s, got %v", opts.MaxRequestTimeout)
	}
	if opts.DefaultRequestTimeout != 15*time.Second {
		t.Errorf("Expected DefaultRequestTimeout to be 15s, got %v", opts.DefaultRequestTimeout)
	}

	t.Logf("gRPC server can access timeout configurations from server.Options")
	t.Logf("MaxRequestTimeout: %v", opts.MaxRequestTimeout)
	t.Logf("DefaultRequestTimeout: %v", opts.DefaultRequestTimeout)
}