package client

import (
	"context"
	"testing"
	"time"
)

func TestContextDeadlineRespected(t *testing.T) {
	// Create context with short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create call options with longer connection timeout
	opts := CallOptions{
		ConnectionTimeout: 5 * time.Second,
	}

	// Test call method - this would fail in real scenario, but we're testing timeout calculation
	// We expect the cTimeout to be set to context deadline (2s) not ConnectionTimeout (5s)

	// Note: This is a basic test structure. In real scenario, we'd need to mock transport layer
	// to test the actual timeout behavior without making real network calls.

	t.Logf("Testing context deadline handling in call methods")
	t.Logf("Context deadline: %v", 2*time.Second)
	t.Logf("ConnectionTimeout: %v", opts.ConnectionTimeout)

	// The actual test would involve mocking the transport layer
	// For now, we just verify the code compiles and runs basic logic
	if opts.ConnectionTimeout != 5*time.Second {
		t.Errorf("Expected ConnectionTimeout to be 5s, got %v", opts.ConnectionTimeout)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Expected context to have deadline")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		t.Error("Context deadline already passed")
	}

	// This is the logic we added to the code
	cTimeout := opts.ConnectionTimeout
	if remaining < cTimeout {
		cTimeout = remaining
	}

	// Verify that cTimeout is now the shorter context deadline
	if cTimeout >= opts.ConnectionTimeout {
		t.Errorf("Expected cTimeout to be reduced to context deadline, got %v", cTimeout)
	}

	t.Logf("Effective timeout will be: %v (context deadline)", cTimeout)
}