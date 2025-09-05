package profile

import (
	"context"
	"net"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"go-micro.dev/v5/client"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/wrapper/trace/opentelemetry"

	"github.com/zigo2048/mcbeam-common-lib/common/metrics"
	metricsWrapper "github.com/zigo2048/mcbeam-common-lib/common/metrics/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/apiheader"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/plugins/prometheus/v3"
)

// ObservabilityConfig holds configuration for observability features
type ObservabilityConfig struct {
	// Tracing configuration
	TracingEnabled  bool
	TracingReporter string
	
	// Metrics configuration  
	MetricsEnabled bool
	MetricsReporter string
	
	// Service information for tracing
	ServiceName    string
	ServiceVersion string
}

// NewObservabilityConfig creates a new ObservabilityConfig with default values from environment
func NewObservabilityConfig() *ObservabilityConfig {
	config := &ObservabilityConfig{
		TracingReporter: os.Getenv("MICRO_TRACING_REPORTER_ADDRESS"),
		ServiceName:     os.Getenv("MICRO_SERVER_NAME"),
		ServiceVersion:  os.Getenv("MICRO_SERVER_VERSION"),
		MetricsEnabled:  true, // Default enable metrics
	}
	
	// Enable tracing if reporter address is configured
	config.TracingEnabled = len(config.TracingReporter) > 0
	
	return config
}

// InitObservability initializes observability components including tracing and metrics
func InitObservability(cli *client.Client, srv *server.Server, config *ObservabilityConfig) error {
	if config == nil {
		config = NewObservabilityConfig()
	}

	// Initialize tracing if enabled and reporter address is configured
	if config.TracingEnabled && len(config.TracingReporter) > 0 {
		if err := initTracing(cli, srv, config); err != nil {
			return err
		}
	}

	// Initialize metrics if enabled
	if config.MetricsEnabled {
		if err := initMetrics(srv, config); err != nil {
			return err
		}
	}

	return nil
}

// EnableObservability is a convenience method for Profile to enable observability
func (p *Profile) EnableObservability(cli *client.Client, srv *server.Server) error {
	if p.Observability == nil {
		p.Observability = NewObservabilityConfig()
	}
	return InitObservability(cli, srv, p.Observability)
}

// initTracing configures OpenTelemetry tracing
func initTracing(cli *client.Client, srv *server.Server, config *ObservabilityConfig) error {
	if len(config.TracingReporter) == 0 {
		logger.Debug("No tracing reporter address configured, skipping tracing setup")
		return nil
	}

	tracer, err := newTracerProvider(config.TracingReporter, config.ServiceName, config.ServiceVersion)
	if err != nil {
		logger.Fatalf("tracer provider error: %s reporterAddress:%s", err.Error(), config.TracingReporter)
		return err
	}

	otel.SetTracerProvider(tracer)

	// Add OpenTelemetry wrappers to server
	if srv != nil {
		err = (*srv).Init(
			server.WrapHandler(opentelemetry.NewHandlerWrapper()),
			server.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
		)
		if err != nil {
			return err
		}
	}

	// Add OpenTelemetry wrapper to client
	if cli != nil {
		err = (*cli).Init(
			client.Wrap(opentelemetry.NewClientWrapper()),
		)
		if err != nil {
			return err
		}
	}

	logger.Infof("Tracing initialized with reporter: %s", config.TracingReporter)
	return nil
}

// initMetrics configures Prometheus metrics
func initMetrics(srv *server.Server, config *ObservabilityConfig) error {
	reporter, err := prometheus.New()
	if err != nil {
		logger.Fatal(err)
		return err
	}

	metrics.SetDefaultMetricsReporter(reporter)

	// Add metrics and other wrappers to default server
	err = server.DefaultServer.Init(
		server.WrapHandler(metricsWrapper.New(reporter).HandlerFunc),
		server.WrapHandler(apiheader.NewDefaultHeaderHandlerWrapper),
		server.WrapHandler(wrapper.AuthHandler()),
	)
	if err != nil {
		logger.Fatalf("init default server err:%s", err)
		return err
	}

	logger.Info("Metrics initialized with Prometheus")
	return nil
}

// newJaegerExporter creates a new Jaeger exporter based on the address format
func newJaegerExporter(ctx context.Context, address string) (trace.SpanExporter, error) {
	var exporter trace.SpanExporter
	var err error

	if strings.HasPrefix(address, "http") {
		// HTTP collector endpoint: http://jaeger-collector.monitoring.svc.cluster.local:14268/api/traces
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(address)))
	} else {
		// UDP agent endpoint: jaeger-agent.monitoring.svc.cluster.local:6831
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		logger.Debugf("jaeger address, host:%s port:%s", host, port)
		exporter, err = jaeger.New(
			jaeger.WithAgentEndpoint(jaeger.WithAgentHost(host), jaeger.WithAgentPort(port)),
		)
	}

	if err != nil {
		logger.Errorf("new Jaeger exporter err:%s", err.Error())
		return nil, err
	}

	return exporter, nil
}

// newTracerProvider creates a new OpenTelemetry TracerProvider
func newTracerProvider(url, serviceName, serviceVersion string) (*trace.TracerProvider, error) {
	ctx := context.Background()

	// Use default service name if not provided
	if len(serviceName) == 0 {
		serviceName = "unknown-service"
	}
	if len(serviceVersion) == 0 {
		serviceVersion = "unknown-version"
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		logger.Fatalf("error creating resource: %v", err)
		return nil, err
	}

	// Create exporter
	exporter, err := newJaegerExporter(ctx, url)
	if err != nil {
		logger.Fatalf("error creating exporter: %v", err)
		return nil, err
	}

	// Set up propagation
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	// Create TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	return tp, nil
}