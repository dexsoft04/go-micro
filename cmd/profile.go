package cmd

import (
	"context"
	"fmt"
	"go-micro.dev/v5/wrapper/trace/opentelemetry"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"net"
	"os"
	"strings"

	fileexporter "go-micro.dev/v5/cmd/exporter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initConfig(ctx *cli.Context) (error, []client.Option, []server.Option) {
	var clientOpts []client.Option
	var serverOpts []server.Option
	reporterAddress := os.Getenv("MICRO_TRACING_REPORTER_ADDRESS")
	if len(reporterAddress) == 0 {
		return nil, clientOpts, serverOpts
	}
	tracer, err := tracerProvider(ctx.Context, reporterAddress)
	if nil != err {
		logger.Errorf("tracer provider error: %s reporterAddress:%s", err.Error(), reporterAddress)
		return err, clientOpts, serverOpts
	}
	otel.SetTracerProvider(tracer)

	clientOpts = append(clientOpts, client.Wrap(opentelemetry.NewClientWrapper()))

	serverOpts = append(serverOpts,
		server.WrapHandler(opentelemetry.NewHandlerWrapper()),
		server.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
	)
	return nil, clientOpts, serverOpts
}

func newExporter(ctx context.Context, address string) (trace.SpanExporter, error) {
	var exporter trace.SpanExporter
	var err error

	// Priority 1: Check for file output mode
	if strings.HasPrefix(address, "file://") {
		// file:///path/to/traces.jsonl - output to file
		filePath := strings.TrimPrefix(address, "file://")

		// Check file format
		format := os.Getenv("MICRO_TRACING_FILE_FORMAT")
		switch format {
		case "simple":
			// Use simplified JSON format for better performance analysis
			exporter, err = fileexporter.NewSimpleFileExporter(filePath)
			if err != nil {
				logger.Errorf("failed to create simple file exporter: %v", err)
				return nil, err
			}
			logger.Infof("Using simple file exporter: %s", filePath)
			return exporter, nil

		case "json":
			// Use standard OpenTelemetry JSON format
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Errorf("failed to open trace file %s: %v", filePath, err)
				return nil, fmt.Errorf("failed to open trace file: %v", err)
			}
			exporter, err = stdouttrace.New(
				stdouttrace.WithWriter(file),
				stdouttrace.WithPrettyPrint(), // JSON format with pretty printing
			)
			if err != nil {
				file.Close()
				logger.Errorf("failed to create JSON file exporter: %v", err)
				return nil, err
			}
			logger.Infof("Using JSON file exporter: %s", filePath)
			return exporter, nil

		default:
			// Default: use standard OpenTelemetry format (compact)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Errorf("failed to open trace file %s: %v", filePath, err)
				return nil, fmt.Errorf("failed to open trace file: %v", err)
			}
			exporter, err = stdouttrace.New(
				stdouttrace.WithWriter(file),
				stdouttrace.WithoutTimestamps(), // use span's own timestamps
			)
			if err != nil {
				file.Close()
				logger.Errorf("failed to create standard file exporter: %v", err)
				return nil, err
			}
			logger.Infof("Using standard file exporter: %s", filePath)
			return exporter, nil
		}
	}

	// Priority 2: stdout mode
	if address == "stdout" {
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			logger.Errorf("failed to create stdout exporter: %v", err)
			return nil, err
		}
		logger.Info("Using stdout exporter")
		return exporter, nil
	}

	// Priority 3: Jaeger mode (keep original logic)
	if strings.HasPrefix(address, "http") {
		//http://jaeger-collector.monitoring.svc.cluster.local:14268/api/traces
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(address)))
		//exporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint(address))
		if err != nil {
			logger.Errorf("failed to create Jaeger HTTP exporter: %v", err)
			return nil, err
		}
		logger.Infof("Using Jaeger HTTP collector: %s", address)
	} else {
		//jaeger-agent.monitoring.svc.cluster.local:6831
		host, port, err := net.SplitHostPort(address)
		if nil != err {
			logger.Errorf("invalid Jaeger agent address %s: %v", address, err)
			return nil, err
		}
		logger.Debugf("jaeger address, host:%s port:%s", host, port)
		exporter, err = jaeger.New(
			jaeger.WithAgentEndpoint(jaeger.WithAgentHost(host), jaeger.WithAgentPort(port)),
		)
		//exporter, err = otlptracegrpc.New(ctx,
		//	otlptracegrpc.WithAgentEndpoint(otlptracegrpc.WithAgentHost(host), otlptracegrpc.WithAgentPort(port)),
		//)
		if err != nil {
			logger.Errorf("failed to create Jaeger UDP exporter: %v", err)
			return nil, err
		}
		logger.Infof("Using Jaeger UDP agent: %s", address)
	}

	return exporter, nil
}

func tracerProvider(ctx context.Context, url string) (*trace.TracerProvider, error) {
	// Get instance ID for multi-instance environments
	instanceID := os.Getenv("MICRO_INSTANCE_ID")
	if instanceID == "" {
		hostname, _ := os.Hostname()
		instanceID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}

	// Check if pressure testing mode is enabled
	pressureTestMode := os.Getenv("MICRO_PRESSURE_TEST_MODE")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(os.Getenv("MICRO_SERVICE_NAME")),
			semconv.ServiceVersion(os.Getenv("MICRO_SERVER_VERSION")),
			attribute.String("instance.id", instanceID),
			attribute.String("pressure.test.mode", pressureTestMode),
		),
	)
	if err != nil {
		logger.Fatalf("error creating resource: %v", err)
	}
	exporter, err := newExporter(ctx, url)
	if nil != err {
		logger.Fatalf("error creating exporter: %v", err)
	}

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	// Configure trace provider options based on pressure testing mode
	var options []trace.TracerProviderOption
	options = append(options, trace.WithResource(res))

	if pressureTestMode == "true" {
		// Pressure testing mode: optimize for performance and completeness
		logger.Info("Pressure testing mode enabled - using optimized batch settings")
		options = append(options, trace.WithBatcher(exporter,
			trace.WithBatchTimeout(100*time.Millisecond), // Fast batch processing
			trace.WithMaxExportBatchSize(512),            // Larger batches
			trace.WithMaxQueueSize(2048),                 // Larger queue
		))
		// Always sample in pressure testing mode
		options = append(options, trace.WithSampler(trace.AlwaysSample()))
	} else {
		// Normal mode: standard settings
		options = append(options, trace.WithBatcher(exporter))
		// Use default sampler (can be configured via OTEL_TRACES_SAMPLER)
	}

	tp := trace.NewTracerProvider(options...)
	return tp, nil
}
