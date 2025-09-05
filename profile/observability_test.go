package profile

import (
	"os"
	"testing"
)

func TestNewObservabilityConfig(t *testing.T) {
	// Test with no environment variables
	config := NewObservabilityConfig()
	if config == nil {
		t.Fatal("NewObservabilityConfig returned nil")
	}
	
	// Should have metrics enabled by default
	if !config.MetricsEnabled {
		t.Error("Expected metrics to be enabled by default")
	}
	
	// Should not have tracing enabled without reporter address
	if config.TracingEnabled {
		t.Error("Expected tracing to be disabled without reporter address")
	}
}

func TestNewObservabilityConfigWithEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("MICRO_TRACING_REPORTER_ADDRESS", "http://jaeger:14268/api/traces")
	os.Setenv("MICRO_SERVER_NAME", "test-service")
	os.Setenv("MICRO_SERVER_VERSION", "1.0.0")
	
	defer func() {
		os.Unsetenv("MICRO_TRACING_REPORTER_ADDRESS")
		os.Unsetenv("MICRO_SERVER_NAME")
		os.Unsetenv("MICRO_SERVER_VERSION")
	}()
	
	config := NewObservabilityConfig()
	
	// Should have tracing enabled with reporter address
	if !config.TracingEnabled {
		t.Error("Expected tracing to be enabled with reporter address")
	}
	
	if config.TracingReporter != "http://jaeger:14268/api/traces" {
		t.Errorf("Expected tracing reporter to be set, got: %s", config.TracingReporter)
	}
	
	if config.ServiceName != "test-service" {
		t.Errorf("Expected service name to be 'test-service', got: %s", config.ServiceName)
	}
	
	if config.ServiceVersion != "1.0.0" {
		t.Errorf("Expected service version to be '1.0.0', got: %s", config.ServiceVersion)
	}
}

func TestLocalProfile(t *testing.T) {
	profile, err := LocalProfile()
	if err != nil {
		t.Fatalf("LocalProfile failed: %v", err)
	}
	
	if profile.Observability == nil {
		t.Fatal("Expected LocalProfile to have Observability config")
	}
	
	// Local profile should have metrics enabled but tracing disabled by default
	if !profile.Observability.MetricsEnabled {
		t.Error("Expected LocalProfile to have metrics enabled")
	}
	
	if profile.Observability.TracingEnabled {
		t.Error("Expected LocalProfile to have tracing disabled by default")
	}
	
	if profile.Observability.ServiceName != "local-service" {
		t.Errorf("Expected service name to be 'local-service', got: %s", profile.Observability.ServiceName)
	}
}

func TestNatsProfile(t *testing.T) {
	// Skip this test if NATS server is not available
	// This test requires a running NATS server
	t.Skip("Skipping NatsProfile test - requires running NATS server")
	
	// Set NATS address to avoid connecting to real NATS server
	os.Setenv("MICRO_NATS_ADDRESS", "nats://localhost:4222")
	defer os.Unsetenv("MICRO_NATS_ADDRESS")
	
	profile, err := NatsProfile()
	if err != nil {
		t.Fatalf("NatsProfile failed: %v", err)
	}
	
	if profile.Observability == nil {
		t.Fatal("Expected NatsProfile to have Observability config")
	}
	
	// NATS profile should have metrics enabled
	if !profile.Observability.MetricsEnabled {
		t.Error("Expected NatsProfile to have metrics enabled")
	}
	
	if profile.Observability.ServiceName != "nats-service" {
		t.Errorf("Expected service name to be 'nats-service', got: %s", profile.Observability.ServiceName)
	}
}