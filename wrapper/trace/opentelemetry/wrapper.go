package opentelemetry

import (
	"context"
	"fmt"
	"time"

	"go-micro.dev/v5/client"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NewCallWrapper accepts an opentracing Tracer and returns a Call Wrapper.
func NewCallWrapper(opts ...Option) client.CallWrapper {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			if options.CallFilter != nil && options.CallFilter(ctx, req) {
				return cf(ctx, node, req, rsp, opts)
			}
			
			// Record start time for detailed timing
			startTime := time.Now()
			
			name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
			spanOpts := []trace.SpanStartOption{
				trace.WithSpanKind(trace.SpanKindClient),
			}
			ctx, span := StartSpanFromContext(ctx, options.TraceProvider, name, spanOpts...)
			
			defer func() {
				// Calculate total duration
				duration := time.Since(startTime)
				
				// Output detailed timing log
				log.Logf(log.InfoLevel, 
					"RPC_TIMING: service=%s endpoint=%s node=%s duration_ms=%d trace_id=%s span_id=%s",
					req.Service(), 
					req.Endpoint(),
					node.Address,
					duration.Milliseconds(),
					span.SpanContext().TraceID().String(),
					span.SpanContext().SpanID().String(),
				)
				
				// Set span attributes
				span.SetAttributes(
					attribute.Int64("rpc.duration_ms", duration.Milliseconds()),
					attribute.String("rpc.node_address", node.Address),
				)
				
				span.End()
			}()
			
			err := cf(ctx, node, req, rsp, opts)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
			}
			return err
		}
	}
}

// NewHandlerWrapper accepts an opentracing Tracer and returns a Handler Wrapper.
func NewHandlerWrapper(opts ...Option) server.HandlerWrapper {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if options.HandlerFilter != nil && options.HandlerFilter(ctx, req) {
				return h(ctx, req, rsp)
			}
			
			// Record start time for detailed timing
			startTime := time.Now()
			
			name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
			spanOpts := []trace.SpanStartOption{
				trace.WithSpanKind(trace.SpanKindServer),
			}
			ctx, span := StartSpanFromContext(ctx, options.TraceProvider, name, spanOpts...)
			
			defer func() {
				// Calculate total duration
				duration := time.Since(startTime)
				
				// Output server timing log
				log.Logf(log.InfoLevel,
					"RPC_TIMING_SERVER: service=%s endpoint=%s duration_ms=%d trace_id=%s span_id=%s",
					req.Service(),
					req.Endpoint(), 
					duration.Milliseconds(),
					span.SpanContext().TraceID().String(),
					span.SpanContext().SpanID().String(),
				)
				
				// Set span attributes
				span.SetAttributes(
					attribute.Int64("rpc.server.duration_ms", duration.Milliseconds()),
				)
				
				span.End()
			}()
			
			err := h(ctx, req, rsp)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
			}
			return err
		}
	}
}

// NewSubscriberWrapper accepts an opentracing Tracer and returns a Subscriber Wrapper.
func NewSubscriberWrapper(opts ...Option) server.SubscriberWrapper {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	return func(next server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, msg server.Message) error {
			if options.SubscriberFilter != nil && options.SubscriberFilter(ctx, msg) {
				return next(ctx, msg)
			}
			name := "Sub from " + msg.Topic()
			spanOpts := []trace.SpanStartOption{
				trace.WithSpanKind(trace.SpanKindServer),
			}
			ctx, span := StartSpanFromContext(ctx, options.TraceProvider, name, spanOpts...)
			defer span.End()
			if err := next(ctx, msg); err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
				return err
			}
			return nil
		}
	}
}

// NewClientWrapper returns a client.Wrapper
// that adds monitoring to outgoing requests.
func NewClientWrapper(opts ...Option) client.Wrapper {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	return func(c client.Client) client.Client {
		w := &clientWrapper{
			Client:        c,
			tp:            options.TraceProvider,
			callFilter:    options.CallFilter,
			streamFilter:  options.StreamFilter,
			publishFilter: options.PublishFilter,
		}
		return w
	}
}

type clientWrapper struct {
	client.Client

	tp            trace.TracerProvider
	callFilter    CallFilter
	streamFilter  StreamFilter
	publishFilter PublishFilter
}

func (w *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if w.callFilter != nil && w.callFilter(ctx, req) {
		return w.Client.Call(ctx, req, rsp, opts...)
	}
	
	// Record start time for detailed timing
	startTime := time.Now()
	
	name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
	}
	ctx, span := StartSpanFromContext(ctx, w.tp, name, spanOpts...)
	
	defer func() {
		// Calculate total duration
		duration := time.Since(startTime)
		
		// Output client wrapper timing log
		log.Logf(log.InfoLevel, 
			"RPC_TIMING_CLIENT: service=%s endpoint=%s duration_ms=%d trace_id=%s span_id=%s",
			req.Service(), 
			req.Endpoint(),
			duration.Milliseconds(),
			span.SpanContext().TraceID().String(),
			span.SpanContext().SpanID().String(),
		)
		
		// Set span attributes
		span.SetAttributes(
			attribute.Int64("rpc.client.duration_ms", duration.Milliseconds()),
		)
		
		span.End()
	}()
	
	err := w.Client.Call(ctx, req, rsp, opts...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

func (w *clientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	if w.streamFilter != nil && w.streamFilter(ctx, req) {
		return w.Client.Stream(ctx, req, opts...)
	}
	name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
	}
	ctx, span := StartSpanFromContext(ctx, w.tp, name, spanOpts...)
	defer span.End()
	stream, err := w.Client.Stream(ctx, req, opts...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return stream, err
}

func (w *clientWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	if w.publishFilter != nil && w.publishFilter(ctx, p) {
		return w.Client.Publish(ctx, p, opts...)
	}
	name := fmt.Sprintf("Pub to %s", p.Topic())
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
	}
	ctx, span := StartSpanFromContext(ctx, w.tp, name, spanOpts...)
	defer span.End()
	if err := w.Client.Publish(ctx, p, opts...); err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}
	return nil
}
