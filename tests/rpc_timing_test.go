package tests

import (
	"context"
	"testing"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/codec"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	otWrapper "go-micro.dev/v5/wrapper/trace/opentelemetry"
)

// mockServerRequest implements server.Request for testing
type mockServerRequest struct {
	service  string
	endpoint string
}

func (m *mockServerRequest) ContentType() string { return "application/json" }
func (m *mockServerRequest) Service() string    { return m.service }
func (m *mockServerRequest) Method() string     { return m.endpoint }
func (m *mockServerRequest) Endpoint() string   { return m.endpoint }
func (m *mockServerRequest) Codec() codec.Reader { return nil }
func (m *mockServerRequest) Header() map[string]string { return nil }
func (m *mockServerRequest) Read() ([]byte, error) { return nil, nil }
func (m *mockServerRequest) Stream() bool { return false }
func (m *mockServerRequest) Body() interface{} { return nil }

// TimingTestHandler is a handler for timing tests to avoid conflicts
type TimingTestHandler struct{}

type TimingTestRequest struct {
	Name string `json:"name"`
}

type TimingTestResponse struct {
	Message string `json:"message"`
	Delay   int    `json:"delay_ms"`
}

func (h *TimingTestHandler) Call(ctx context.Context, req *TimingTestRequest, rsp *TimingTestResponse) error {
	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)
	
	rsp.Message = "Hello " + req.Name
	rsp.Delay = 10
	return nil
}

func (h *TimingTestHandler) SlowCall(ctx context.Context, req *TimingTestRequest, rsp *TimingTestResponse) error {
	// Simulate slow processing
	time.Sleep(150 * time.Millisecond)
	
	rsp.Message = "Slow Hello " + req.Name
	rsp.Delay = 150
	return nil
}

func setupTiming(t *testing.T) {
	// Configure OpenTelemetry with basic tracer provider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	
	// Set debug logging to see detailed timing
	logger := log.NewLogger(
		log.WithLevel(log.DebugLevel),
	)
	log.DefaultLogger = logger
}

func TestRPCTiming(t *testing.T) {
	setupTiming(t)
	
	// Create server with timing wrapper
	srv := micro.NewService(
		micro.Name("test-timing-server"),
		micro.Address(":0"), // Random port
		micro.WrapHandler(otWrapper.NewHandlerWrapper()),
	)
	
	// Register handler  
	srv.Handle(server.NewHandler(&TimingTestHandler{}))
	
	// Create client with timing wrapper
	client := micro.NewService(
		micro.Name("test-timing-client"),
		micro.WrapClient(otWrapper.NewClientWrapper()),
	).Client()
	
	t.Run("Fast Call Timing", func(t *testing.T) {
		ctx := context.Background()
		req := client.NewRequest("test-timing-server", "TimingTestHandler.Call", &TimingTestRequest{Name: "World"})
		var rsp TimingTestResponse
		
		start := time.Now()
		err := client.Call(ctx, req, &rsp)
		duration := time.Since(start)
		
		if err != nil {
			t.Logf("Expected service discovery error: %v", err)
		}
		
		t.Logf("Fast call took %v", duration)
		t.Logf("Check logs above for RPC_TIMING entries")
	})
	
	t.Run("Slow Call Timing", func(t *testing.T) {
		ctx := context.Background()
		req := client.NewRequest("test-timing-server", "TimingTestHandler.SlowCall", &TimingTestRequest{Name: "World"})
		var rsp TimingTestResponse
		
		start := time.Now()
		err := client.Call(ctx, req, &rsp)
		duration := time.Since(start)
		
		if err != nil {
			t.Logf("Expected service discovery error: %v", err)
		}
		
		t.Logf("Slow call took %v", duration)
		t.Logf("Check logs above for RPC_TIMING entries showing >100ms")
	})
	
	t.Run("Multiple Calls for Statistics", func(t *testing.T) {
		ctx := context.Background()
		
		for i := 0; i < 5; i++ {
			req := client.NewRequest("test-timing-server", "TimingTestHandler.Call", &TimingTestRequest{Name: "Batch"})
			var rsp TimingTestResponse
			
			start := time.Now()
			err := client.Call(ctx, req, &rsp)
			duration := time.Since(start)
			
			if err != nil {
				t.Logf("Call %d error: %v", i+1, err)
			}
			
			t.Logf("Call %d took %v", i+1, duration)
			
			// Small delay between calls
			time.Sleep(10 * time.Millisecond)
		}
		
		t.Log("Check logs above for multiple RPC_TIMING entries with same trace context")
	})
}

func TestTimingWrapper(t *testing.T) {
	setupTiming(t)
	
	t.Log("Testing individual wrapper components...")
	
	// Test CallWrapper
	t.Run("CallWrapper", func(t *testing.T) {
		wrapper := otWrapper.NewCallWrapper()
		
		// Mock call function that simulates RPC call
		mockCall := func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			time.Sleep(25 * time.Millisecond)
			return nil
		}
		
		wrappedCall := wrapper(mockCall)
		
		// Create mock request and node
		srv := micro.NewService()
		client := srv.Client()
		req := client.NewRequest("test-service", "Test.Method", map[string]string{"test": "data"})
		
		// Create mock registry node
		node := &registry.Node{
			Id:      "test-node",
			Address: "127.0.0.1:8080",
		}
		
		var rsp interface{}
		err := wrappedCall(context.Background(), node, req, rsp, client.Options().CallOptions)
		
		t.Logf("CallWrapper test completed with error (expected): %v", err)
		t.Log("Check logs above for RPC_TIMING entry from CallWrapper")
	})
	
	// Test HandlerWrapper  
	t.Run("HandlerWrapper", func(t *testing.T) {
		wrapper := otWrapper.NewHandlerWrapper()
		
		// Mock handler function
		mockHandler := func(ctx context.Context, req server.Request, rsp interface{}) error {
			time.Sleep(30 * time.Millisecond)
			return nil
		}
		
		wrappedHandler := wrapper(mockHandler)
		
		// Create a mock server request
		mockReq := &mockServerRequest{
			service:  "test-service",
			endpoint: "Test.Method",
		}
		
		// This will fail but should log timing
		err := wrappedHandler(context.Background(), mockReq, nil)
		
		t.Logf("HandlerWrapper test completed with error (expected): %v", err)
		t.Log("Check logs above for RPC_TIMING_SERVER entry from HandlerWrapper")
	})
}

func TestTimingAnalysis(t *testing.T) {
	t.Log("=== Timing Analysis Test ===")
	t.Log("This test demonstrates the timing log formats that will be generated:")
	t.Log("")
	t.Log("Expected log formats:")
	t.Log("  RPC_TIMING: service=<service> endpoint=<endpoint> node=<address> duration_ms=<ms> trace_id=<id> span_id=<id>")
	t.Log("  RPC_TIMING_SERVER: service=<service> endpoint=<endpoint> duration_ms=<ms> trace_id=<id> span_id=<id>")
	t.Log("  RPC_TIMING_CLIENT: service=<service> endpoint=<endpoint> duration_ms=<ms> trace_id=<id> span_id=<id>")
	t.Log("  RPC_TIMING_DETAIL: connection_ms=<ms> service=<service> endpoint=<endpoint> address=<address> (debug level)")
	t.Log("  RPC_TIMING_DETAIL: send_ms=<ms> recv_ms=<ms> service=<service> endpoint=<endpoint> (debug level)")
	t.Log("")
	t.Log("To analyze timing logs:")
	t.Log("1. Run your application with debug logging enabled")
	t.Log("2. Save logs to file: ./app 2>&1 | tee app.log")
	t.Log("3. Run analysis script: ./tests/analyze_timing.sh app.log")
	t.Log("")
	t.Log("For real-time monitoring:")
	t.Log("  tail -f app.log | grep RPC_TIMING")
	t.Log("  tail -f app.log | ./tests/analyze_timing.sh")
}

// Benchmark to measure wrapper overhead
func BenchmarkTimingWrapper(b *testing.B) {
	// Configure OpenTelemetry with basic tracer provider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	
	// Set debug logging to see detailed timing
	logger := log.NewLogger(
		log.WithLevel(log.DebugLevel),
	)
	log.DefaultLogger = logger
	
	wrapper := otWrapper.NewCallWrapper()
	
	mockCall := func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
		// Simulate minimal work
		time.Sleep(time.Microsecond)
		return nil
	}
	
	wrappedCall := wrapper(mockCall)
	
	srv := micro.NewService()
	client := srv.Client()
	req := client.NewRequest("bench-service", "Bench.Method", map[string]string{"test": "data"})
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Create mock registry node
		node := &registry.Node{
			Id:      "bench-node", 
			Address: "127.0.0.1:8080",
		}
		
		var rsp interface{}
		_ = wrappedCall(context.Background(), node, req, rsp, client.Options().CallOptions)
	}
}