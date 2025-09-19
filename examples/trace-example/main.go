package main

import (
	"context"
	"log"
	"os"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/server"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
	// Simulate some work
	time.Sleep(50 * time.Millisecond)
	rsp.Greeting = "Hello " + req.Name
	return nil
}

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Greeting string `json:"greeting"`
}

func main() {
	// Set up tracing to file for testing
	os.Setenv("MICRO_TRACING_REPORTER_ADDRESS", "file:///tmp/example-traces.jsonl")
	os.Setenv("MICRO_TRACING_FILE_FORMAT", "simple")
	os.Setenv("MICRO_SERVICE_NAME", "greeter-service")
	os.Setenv("MICRO_SERVER_VERSION", "1.0.0")

	// Create service
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Version("latest"),
	)

	// Register handler
	server.Handle(
		server.NewHandler(&Greeter{}),
	)

	// Initialize service
	service.Init()

	log.Println("Starting greeter service with trace file output...")
	log.Println("Trace output: /tmp/example-traces.jsonl")

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}