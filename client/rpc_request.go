package client

import (
	"github.com/golang/protobuf/proto"
	"go-micro.dev/v5/codec"
	"go-micro.dev/v5/logger"
	goproto "google.golang.org/protobuf/proto"
)

type rpcRequest struct {
	opts        RequestOptions
	codec       codec.Codec
	body        interface{}
	service     string
	method      string
	endpoint    string
	contentType string
}

func newRequest(service, endpoint string, request interface{}, contentType string, reqOpts ...RequestOption) Request {
	var opts RequestOptions

	for _, o := range reqOpts {
		o(&opts)
	}

	// set the content-type specified
	if len(opts.ContentType) > 0 {
		contentType = opts.ContentType
	}
	logger.Tracef("=============== newRequest contentType:%v %T", contentType, request)
	if len(contentType) == 0 {
		if _, ok := request.(proto.Message); ok {
			contentType = "application/protobuf"
		} else if _, ok = request.(goproto.Message); ok {
			contentType = "application/protobuf"
		} else {
			contentType = "application/json"
		}
		logger.Tracef("=============== newRequest set default contentType:%v %T", contentType, request)
	}

	return &rpcRequest{
		service:     service,
		method:      endpoint,
		endpoint:    endpoint,
		body:        request,
		contentType: contentType,
		opts:        opts,
	}
}

func (r *rpcRequest) ContentType() string {
	return r.contentType
}

func (r *rpcRequest) Service() string {
	return r.service
}

func (r *rpcRequest) Method() string {
	return r.method
}

func (r *rpcRequest) Endpoint() string {
	return r.endpoint
}

func (r *rpcRequest) Body() interface{} {
	return r.body
}

func (r *rpcRequest) Codec() codec.Writer {
	return r.codec
}

func (r *rpcRequest) Stream() bool {
	return r.opts.Stream
}
