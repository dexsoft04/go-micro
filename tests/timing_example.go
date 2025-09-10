// Build with: go build -o timing_example timing_example.go
//go:build ignore

package main

import (
	"context"
	"time"

	"go-micro.dev/v5"
	log "go-micro.dev/v5/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	otWrapper "go-micro.dev/v5/wrapper/trace/opentelemetry"
)

// Example configuration for enabling RPC timing logging
func main() {
	// 1. Configure OpenTelemetry for tracing (optional - for stdout debugging)
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal("failed to create stdout exporter:", err)
	}
	
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	
	// 2. Set logger level to Debug to see detailed timing information
	logger := log.NewLogger(
		log.WithLevel(log.DebugLevel),
		log.WithOutput(log.NewHelper(log.DefaultLogger).Out),
	)
	log.DefaultLogger = logger
	
	// 3. Create service with OpenTelemetry wrappers
	service := micro.NewService(
		micro.Name("timing-example-service"),
		micro.Version("v1.0.0"),
		
		// Enable client-side timing wrapper
		micro.WrapClient(otWrapper.NewClientWrapper()),
		
		// Enable server-side timing wrapper  
		micro.WrapHandler(otWrapper.NewHandlerWrapper()),
		
		// Alternative: use CallWrapper for more granular control
		// micro.WrapCall(otWrapper.NewCallWrapper()),
	)
	
	service.Init()
	
	log.Info("Service configured with RPC timing logging")
	log.Info("Timing logs will be output with the following formats:")
	log.Info("  RPC_TIMING: service=<service> endpoint=<endpoint> node=<address> duration_ms=<ms> trace_id=<id> span_id=<id>")
	log.Info("  RPC_TIMING_SERVER: service=<service> endpoint=<endpoint> duration_ms=<ms> trace_id=<id> span_id=<id>") 
	log.Info("  RPC_TIMING_CLIENT: service=<service> endpoint=<endpoint> duration_ms=<ms> trace_id=<id> span_id=<id>")
	log.Info("  RPC_TIMING_DETAIL: connection_ms=<ms> service=<service> endpoint=<endpoint> address=<address> (debug level)")
	log.Info("  RPC_TIMING_DETAIL: send_ms=<ms> recv_ms=<ms> service=<service> endpoint=<endpoint> (debug level)")
	
	// Example client call to demonstrate timing
	client := service.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Make a sample request (this will fail but will log timing)
	req := client.NewRequest("example-service", "Example.Call", map[string]string{"test": "data"})
	var rsp map[string]interface{}
	
	log.Info("Making example RPC call (expected to fail but will show timing logs)...")
	err = client.Call(ctx, req, &rsp)
	if err != nil {
		log.Infof("Expected error (no service running): %v", err)
	}
	
	log.Info("Example completed. Check logs above for timing information.")
}