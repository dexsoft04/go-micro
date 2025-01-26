package cmd

import (
	"context"
	"github.com/micro/plugins/v5/wrapper/trace/opentelemetry"
	"github.com/urfave/cli/v2"
	"github.com/zigo2048/mcbeam-common-lib/common/metrics"
	metricsWrapper "github.com/zigo2048/mcbeam-common-lib/common/metrics/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/apiheader"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/plugins/prometheus/v3"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"net"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initConfig(ctx *cli.Context, cli *client.Client, srv *server.Server) error {
	reporterAddress := os.Getenv("MICRO_TRACING_REPORTER_ADDRESS")
	if len(reporterAddress) > 0 {
		tracer, err := tracerProvider(reporterAddress)
		if nil != err {
			logger.Fatalf("tracer provider error: %s reporterAddress:%s", err.Error(), reporterAddress)
		}
		otel.SetTracerProvider(tracer)
		err = (*srv).Init(
			server.WrapHandler(opentelemetry.NewHandlerWrapper()),
			server.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
		)
		if nil != err {
			return err
		}
	}

	reporter, err := prometheus.New()
	if nil != err {
		logger.Fatal(err)
	}
	metrics.SetDefaultMetricsReporter(reporter)

	err = server.DefaultServer.Init(
		server.WrapHandler(metricsWrapper.New(reporter).HandlerFunc),
		server.WrapHandler(apiheader.NewDefaultHeaderHandlerWrapper),
		server.WrapHandler(wrapper.AuthHandler()),
	)
	if nil != err {
		logger.Fatalf("init default server err:%s", err)
	}
	return nil
}

func newExporter(ctx context.Context, address string) (trace.SpanExporter, error) {
	var exporter trace.SpanExporter
	var err error
	if strings.HasPrefix(address, "http") {
		//http://jaeger-collector.monitoring.svc.cluster.local:14268/api/traces
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(address)))
		//exporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint(address))
	} else {
		//jaeger-agent.monitoring.svc.cluster.local:6831
		host, port, err := net.SplitHostPort(address)
		if nil != err {
			return nil, err
		}
		logger.Debugf("jaeger address, host:%s port:%s", host, port)
		exporter, err = jaeger.New(
			jaeger.WithAgentEndpoint(jaeger.WithAgentHost(host), jaeger.WithAgentPort(port)),
		)
		//exporter, err = otlptracegrpc.New(ctx,
		//	otlptracegrpc.WithAgentEndpoint(otlptracegrpc.WithAgentHost(host), otlptracegrpc.WithAgentPort(port)),
		//)
	}
	if err != nil {
		logger.Errorf("new Exporter err:%s", err.Error())
		return nil, err
	}
	return exporter, nil
}

func tracerProvider(url string) (*trace.TracerProvider, error) {
	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(os.Getenv("MICRO_SERVICE_NAME")),
			semconv.ServiceVersion(os.Getenv("MICRO_SERVER_VERSION")),
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
	// 创建TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	return tp, nil
}
