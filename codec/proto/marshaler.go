package proto

import (
	"bytes"
	"go-micro.dev/v5/logger"
	"runtime/debug"

	"github.com/golang/protobuf/proto"
	"github.com/oxtoacart/bpool"
	"go-micro.dev/v5/codec"
)

// create buffer pool with 16 instances each preallocated with 256 bytes.
var bufferPool = bpool.NewSizedBufferPool(16, 256)

type Marshaler struct{}

func (Marshaler) Marshal(v interface{}) ([]byte, error) {
	logger.Tracef("proto Marshal %T %s", v, string(debug.Stack()))
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, codec.ErrInvalidMessage
	}

	// looks not good, but allows to reuse underlining bytes
	buf := bufferPool.Get()
	pbuf := proto.NewBuffer(buf.Bytes())
	defer func() {
		bufferPool.Put(bytes.NewBuffer(pbuf.Bytes()))
	}()

	if err := pbuf.Marshal(pb); err != nil {
		return nil, err
	}

	return pbuf.Bytes(), nil
}

func (Marshaler) Unmarshal(data []byte, v interface{}) error {
	logger.Tracef("proto Unmarshal %T %s", v, string(debug.Stack()))

	pb, ok := v.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}

	return proto.Unmarshal(data, pb)
}

func (Marshaler) String() string {
	return "proto"
}
