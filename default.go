package micro

import (
	"context"
	"github.com/micro/plugins/v5/wrapper/trace/opentelemetry"
	"github.com/philchia/agollo/v4"
	"github.com/zigo2048/mcbeam-common-lib/common/config"
	"github.com/zigo2048/mcbeam-common-lib/common/metrics"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/apiheader"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/debug"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/plugins/config/apollo/v3"
	"github.com/zigo2048/mcbeam-common-lib/plugins/prometheus/v3"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"os"
	"path/filepath"

	_ "github.com/micro/plugins/v5/broker/nats"
	_ "github.com/micro/plugins/v5/registry/etcd"
	metricsWrapper "github.com/zigo2048/mcbeam-common-lib/common/metrics/wrapper"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

func initDefaultConfig() {
	config.DefaultConfig = apollo.NewConfig(apollo.WithConfig(&agollo.Conf{
		AppID:          os.Getenv("MICRO_NAMESPACE"),
		Cluster:        "default",
		NameSpaceNames: []string{os.Getenv("MICRO_SERVICE_NAME") + ".yaml"},
		MetaAddr:       os.Getenv("MICRO_CONFIG_ADDRESS"),
		CacheDir:       filepath.Join(os.TempDir(), "apollo"),
	}))

	reporter, err := prometheus.New()
	if nil != err {
		logger.Fatal(err)
	}
	metrics.SetDefaultMetricsReporter(reporter)

	reporterAddress := os.Getenv("MICRO_TRACING_REPORTER_ADDRESS")
	tracer, err := tracerProvider(reporterAddress)
	if nil != err {
		logger.Fatalf("tracer provider error: %s reporterAddress:%s", err.Error(), reporterAddress)
	}
	otel.SetTracerProvider(tracer)

	err = server.DefaultServer.Init(
		server.WrapHandler(opentelemetry.NewHandlerWrapper()),
		server.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
		server.WrapHandler(debug.WrapperHandler),
		server.WrapHandler(metricsWrapper.New(reporter).HandlerFunc),
		server.WrapHandler(apiheader.NewDefaultHeaderHandlerWrapper),
		server.WrapHandler(wrapper.AuthHandler()),
	)
	if nil != err {
		logger.Fatalf("init default server err:%s", err)
	}

	client.DefaultClient.Init(
		client.Wrap(opentelemetry.NewClientWrapper()),
	)
}

func newExporter(ctx context.Context, address string) (trace.SpanExporter, error) {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(address),
	)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}
	return exporter, nil
}

func tracerProvider(url string) (*trace.TracerProvider, error) {
	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", os.Getenv("MICRO_SERVICE_NAME")),
		),
	)
	if err != nil {
		logger.Fatalf("error creating resource: %v", err)
	}
	logger.Infof("jaeger url:%s", url)
	exporter, err := newExporter(ctx, url)
	if nil != err {
		logger.Fatalf("error creating exporter: %v", err)
	}
	// 创建TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	return tp, nil
}
