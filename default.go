package micro

import (
	"context"
	"github.com/micro/plugins/v5/wrapper/trace/opentelemetry"
	"github.com/philchia/agollo/v4"
	"github.com/zigo2048/mcbeam-common-lib/common/config"
	"github.com/zigo2048/mcbeam-common-lib/common/metrics"
	metricsWrapper "github.com/zigo2048/mcbeam-common-lib/common/metrics/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/apiheader"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/plugins/config/apollo/v3"
	"github.com/zigo2048/mcbeam-common-lib/plugins/prometheus/v3"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"net"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	_ "github.com/micro/plugins/v5/broker/nats"
	_ "github.com/micro/plugins/v5/registry/etcd"
	_ "go-micro.dev/v5/transport/grpc"
)

func initDefaultConfig() {
	config.DefaultConfig = apollo.NewConfig(apollo.WithConfig(&agollo.Conf{
		AppID:          os.Getenv("MICRO_NAMESPACE"),
		Cluster:        "default",
		NameSpaceNames: []string{os.Getenv("MICRO_SERVICE_NAME") + ".yaml"},
		MetaAddr:       os.Getenv("MICRO_CONFIG_ADDRESS"),
		CacheDir:       filepath.Join(os.TempDir(), "apollo"),
	}))

	reporterAddress := os.Getenv("MICRO_TRACING_REPORTER_ADDRESS")
	if len(reporterAddress) > 0 {
		tracer, err := tracerProvider(reporterAddress)
		if nil != err {
			logger.Fatalf("tracer provider error: %s reporterAddress:%s", err.Error(), reporterAddress)
		}
		otel.SetTracerProvider(tracer)
		server.DefaultServer.Init(
			server.WrapHandler(opentelemetry.NewHandlerWrapper()),
			server.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
		)
		client.DefaultClient.Init(
			client.Wrap(opentelemetry.NewClientWrapper()),
		)
	}

	reporter, err := prometheus.New()
	if nil != err {
		logger.Fatal(err)
	}
	metrics.SetDefaultMetricsReporter(reporter)

	err = server.DefaultServer.Init(
		//server.WrapHandler(debug.WrapperHandler),
		server.WrapHandler(metricsWrapper.New(reporter).HandlerFunc),
		server.WrapHandler(apiheader.NewDefaultHeaderHandlerWrapper),
		server.WrapHandler(wrapper.AuthHandler()),
	)
	if nil != err {
		logger.Fatalf("init default server err:%s", err)
	}
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
		logger.Infof("jaeger address, host:%s port:%s", host, port)
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
