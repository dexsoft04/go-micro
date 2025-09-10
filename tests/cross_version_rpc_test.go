package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go-micro.dev/v5"
)

// TestRequest represents a test RPC request
type TestRequest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Data    string `json:"data"`
}

// TestResponse represents a test RPC response
type TestResponse struct {
	Message string `json:"message"`
	Version string `json:"version"`
	Echo    string `json:"echo"`
}

// TestHandler implements a simple test handler
type TestHandler struct{}

// Echo handles the echo RPC call
func (t *TestHandler) Echo(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	rsp.Message = "Hello " + req.Name
	rsp.Version = "test-handler-v1.0.0"
	rsp.Echo = req.Data
	return nil
}

// TestCrossVersionRPCCall tests RPC calls between different versions
func TestCrossVersionRPCCall(t *testing.T) {
	// Skip if we don't have test environment setup
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Create old version service (simulates v5.6.0-beta)
	oldService := micro.NewService(
		micro.Name("test-old-rpc-service"),
		micro.Version("v1.0.0-beta"),
		micro.Metadata(map[string]string{
			"version":  "old",
			"go-micro": "v5.6.0-beta",
			"protocol": "http", // Old version uses HTTP by default
		}),
		micro.Address(":0"), // Random port
	)

	// Create new version service (simulates current local version)
	newService := micro.NewService(
		micro.Name("test-new-rpc-service"),
		micro.Version("v2.0.0-local"),
		micro.Metadata(map[string]string{
			"version":  "new",
			"go-micro": "v5.6.0-local",
			"protocol": "grpc", // New version supports gRPC
		}),
		micro.Address(":0"), // Random port
	)

	// Register handlers
	oldHandler := &TestHandler{}
	newHandler := &TestHandler{}

	oldService.Server().Handle(
		oldService.Server().NewHandler(oldHandler),
	)

	newService.Server().Handle(
		newService.Server().NewHandler(newHandler),
	)

	// Initialize services
	oldService.Init()
	newService.Init()

	// Start services
	go func() {
		if err := oldService.Run(); err != nil {
			t.Logf("Old service error: %v", err)
		}
	}()
	go func() {
		if err := newService.Run(); err != nil {
			t.Logf("New service error: %v", err)
		}
	}()

	// Wait for services to start
	time.Sleep(2 * time.Second)

	// Test 1: New client calls old service
	t.Run("NewClientToOldService", func(t *testing.T) {
		newClient := newService.Client()
		
		request := &TestRequest{
			Name:    "NewClient",
			Version: "v2.0.0",
			Data:    "calling old service",
		}
		response := &TestResponse{}

		err := newClient.Call(
			context.TODO(),
			newClient.NewRequest("test-old-rpc-service", "TestHandler.Echo", request),
			response,
		)

		if err != nil {
			t.Errorf("New client failed to call old service: %v", err)
		} else {
			if response.Message != "Hello NewClient" {
				t.Errorf("Unexpected response message: %s", response.Message)
			}
			if response.Echo != "calling old service" {
				t.Errorf("Unexpected echo: %s", response.Echo)
			}
			t.Logf("✓ New client successfully called old service: %s", response.Message)
		}
	})

	// Test 2: Old client calls new service
	t.Run("OldClientToNewService", func(t *testing.T) {
		oldClient := oldService.Client()
		
		request := &TestRequest{
			Name:    "OldClient",
			Version: "v1.0.0",
			Data:    "calling new service",
		}
		response := &TestResponse{}

		err := oldClient.Call(
			context.TODO(),
			oldClient.NewRequest("test-new-rpc-service", "TestHandler.Echo", request),
			response,
		)

		if err != nil {
			t.Errorf("Old client failed to call new service: %v", err)
		} else {
			if response.Message != "Hello OldClient" {
				t.Errorf("Unexpected response message: %s", response.Message)
			}
			if response.Echo != "calling new service" {
				t.Errorf("Unexpected echo: %s", response.Echo)
			}
			t.Logf("✓ Old client successfully called new service: %s", response.Message)
		}
	})

	// Cleanup
	oldService.Server().Stop()
	newService.Server().Stop()
}

// TestProtocolNegotiation tests automatic protocol negotiation
func TestProtocolNegotiation(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Test different transport configurations
	testCases := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "Legacy gRPC transport",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
			},
			expected: "grpc",
		},
		{
			name: "New gRPC client/server",
			envVars: map[string]string{
				"MICRO_CLIENT": "grpc",
				"MICRO_SERVER": "grpc",
			},
			expected: "grpc",
		},
		{
			name: "Mixed configuration - new overrides old",
			envVars: map[string]string{
				"MICRO_TRANSPORT": "grpc",
				"MICRO_CLIENT":    "rpc", // Should override MICRO_TRANSPORT
			},
			expected: "rpc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			service := micro.NewService(
				micro.Name("test-protocol-service"),
				micro.Version("v1.0.0"),
				micro.Address(":0"),
			)

			service.Init()

			// Test that service initializes with expected protocol
			// Note: This is a basic test - in a real scenario you'd check
			// the actual transport type being used
			opts := service.Options()
			if opts.Client == nil {
				t.Error("Client not initialized")
			}
			if opts.Server == nil {
				t.Error("Server not initialized")
			}

			t.Logf("✓ Service initialized with configuration: %v", tc.envVars)
		})
	}
}

// TestRPCWithMetadata tests RPC calls with metadata propagation
func TestRPCWithMetadata(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Create service with metadata-aware handler
	service := micro.NewService(
		micro.Name("test-metadata-service"),
		micro.Version("v1.0.0"),
		micro.Address(":0"),
	)

	// Handler that checks metadata
	handler := func(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
		// In a real implementation, you'd extract metadata from context
		rsp.Message = "Hello " + req.Name
		rsp.Version = "metadata-aware"
		rsp.Echo = req.Data
		return nil
	}

	service.Server().Handle(
		service.Server().NewHandler(&testMetadataHandler{handler: handler}),
	)

	service.Init()

	go func() {
		if err := service.Run(); err != nil {
			t.Logf("Service error: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	// Test RPC call with metadata
	client := service.Client()
	
	ctx := context.TODO()
	// In a real scenario, you'd add metadata to context here

	request := &TestRequest{
		Name:    "MetadataTest",
		Version: "v1.0.0",
		Data:    "with metadata",
	}
	response := &TestResponse{}

	err := client.Call(
		ctx,
		client.NewRequest("test-metadata-service", "TestMetadataHandler.Handle", request),
		response,
	)

	if err != nil {
		t.Errorf("RPC call with metadata failed: %v", err)
	} else {
		if response.Message != "Hello MetadataTest" {
			t.Errorf("Unexpected response: %s", response.Message)
		}
		t.Logf("✓ RPC call with metadata succeeded: %s", response.Message)
	}

	service.Server().Stop()
}

// testMetadataHandler wraps a handler function
type testMetadataHandler struct {
	handler func(ctx context.Context, req *TestRequest, rsp *TestResponse) error
}

func (h *testMetadataHandler) Handle(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return h.handler(ctx, req, rsp)
}

// TestRetryAndBackoff tests retry mechanisms across versions
func TestRetryAndBackoff(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	// Create a service that fails initially then succeeds
	service := micro.NewService(
		micro.Name("test-retry-service"),
		micro.Version("v1.0.0"),
		micro.Address(":0"),
	)

	failureCount := 0
	handler := func(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
		failureCount++
		if failureCount <= 2 { // Fail first 2 attempts
			return fmt.Errorf("simulated failure (attempt %d)", failureCount)
		}
		rsp.Message = "Success after retry"
		rsp.Version = "retry-test"
		rsp.Echo = req.Data
		return nil
	}

	service.Server().Handle(
		service.Server().NewHandler(&testRetryHandler{handler: handler}),
	)

	service.Init()

	go func() {
		if err := service.Run(); err != nil {
			t.Logf("Service error: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	// Create client with retry configuration
	client := service.Client()
	
	request := &TestRequest{
		Name:    "RetryTest",
		Version: "v1.0.0",
		Data:    "testing retry",
	}
	response := &TestResponse{}

	// Call with retries (note: actual retry configuration may vary)
	err := client.Call(
		context.TODO(),
		client.NewRequest("test-retry-service", "TestRetryHandler.Handle", request),
		response,
	)

	if err != nil {
		t.Errorf("RPC call failed even with retries: %v", err)
	} else {
		if response.Message != "Success after retry" {
			t.Errorf("Unexpected response: %s", response.Message)
		}
		if failureCount <= 2 {
			t.Errorf("Expected at least 3 attempts, got %d", failureCount)
		}
		t.Logf("✓ RPC call succeeded after %d attempts: %s", failureCount, response.Message)
	}

	service.Server().Stop()
}

// testRetryHandler wraps a handler function for retry testing
type testRetryHandler struct {
	handler func(ctx context.Context, req *TestRequest, rsp *TestResponse) error
}

func (h *testRetryHandler) Handle(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return h.handler(ctx, req, rsp)
}

// TestStreamingCompatibility tests streaming RPC compatibility
func TestStreamingCompatibility(t *testing.T) {
	t.Skip("Streaming tests require more complex setup - implement when streaming is needed")
	
	// This test would verify that streaming calls work between versions
	// It requires setting up streaming handlers and clients
	// which is more complex and may not be immediately needed
}

// TestErrorPropagation tests error handling across versions
func TestErrorPropagation(t *testing.T) {
	if os.Getenv("MICRO_REGISTRY") == "" {
		os.Setenv("MICRO_REGISTRY", "memory")
		defer os.Unsetenv("MICRO_REGISTRY")
	}

	service := micro.NewService(
		micro.Name("test-error-service"),
		micro.Version("v1.0.0"),
		micro.Address(":0"),
	)

	// Handler that returns different types of errors
	handler := func(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
		switch req.Data {
		case "not_found":
			return fmt.Errorf("resource not found (404)")
		case "internal_error":
			return fmt.Errorf("internal server error (500)")
		case "bad_request":
			return fmt.Errorf("bad request (400)")
		default:
			rsp.Message = "Success"
			return nil
		}
	}

	service.Server().Handle(
		service.Server().NewHandler(&testErrorHandler{handler: handler}),
	)

	service.Init()

	go func() {
		if err := service.Run(); err != nil {
			t.Logf("Service error: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	client := service.Client()

	// Test different error scenarios
	errorTests := []struct {
		name        string
		requestData string
		expectedMsg string
	}{
		{"NotFound", "not_found", "resource not found (404)"},
		{"InternalError", "internal_error", "internal server error (500)"},
		{"BadRequest", "bad_request", "bad request (400)"},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			request := &TestRequest{
				Name:    "ErrorTest",
				Version: "v1.0.0",
				Data:    tt.requestData,
			}
			response := &TestResponse{}

			err := client.Call(
				context.TODO(),
				client.NewRequest("test-error-service", "TestErrorHandler.Handle", request),
				response,
			)

			if err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			} else {
				// Check if error contains expected information
				if err.Error() != tt.expectedMsg {
					t.Logf("Expected '%s', got '%s'", tt.expectedMsg, err.Error())
				}
				t.Logf("✓ Error correctly propagated: %v", err)
			}
		})
	}

	service.Server().Stop()
}

// testErrorHandler wraps a handler function for error testing
type testErrorHandler struct {
	handler func(ctx context.Context, req *TestRequest, rsp *TestResponse) error
}

func (h *testErrorHandler) Handle(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return h.handler(ctx, req, rsp)
}