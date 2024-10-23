// Package json provides a json codec
package json

import (
	"encoding/json"
	"go-micro.dev/v5/logger"
	"io"
	"runtime/debug"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go-micro.dev/v5/codec"
)

type Codec struct {
	Conn    io.ReadWriteCloser
	Encoder *json.Encoder
	Decoder *json.Decoder
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	logger.Tracef("json ReadHeader %v t%v %s", m.Type, t, string(debug.Stack()))

	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	if b == nil {
		logger.Tracef("json ReadBody nil b %v", b)
		return nil
	}
	if pb, ok := b.(proto.Message); ok {
		logger.Tracef("jsonpb ReadBody %T %s", b, string(debug.Stack()))
		marshaller := jsonpb.Unmarshaler{AllowUnknownFields: true}
		return marshaller.UnmarshalNext(c.Decoder, pb)
	}
	logger.Tracef("json ReadBody %T", b)
	return c.Decoder.Decode(b)
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	if b == nil {
		return nil
	}

	if jb, ok := b.(*json.RawMessage); ok {
		xx, err := jb.MarshalJSON()
		logger.Tracef("json MarshalJSON %v %v", string(xx), err)
	}
	logger.Tracef("json Write %T %s", b, string(debug.Stack()))
	return c.Encoder.Encode(b)
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "json"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn:    c,
		Decoder: json.NewDecoder(c),
		Encoder: json.NewEncoder(c),
	}
}
