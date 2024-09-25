package json

import (
	"bytes"
	"encoding/json"
	"go-micro.dev/v5/logger"
	"runtime/debug"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/oxtoacart/bpool"
)

var jsonpbMarshaler = &jsonpb.Marshaler{}

// create buffer pool with 16 instances each preallocated with 256 bytes.
var bufferPool = bpool.NewSizedBufferPool(16, 256)

type Marshaler struct{}

func (j Marshaler) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		logger.Debugf("jsonpbMarshaler Marshal %T %s", v, string(debug.Stack()))

		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		if err := jsonpbMarshaler.Marshal(buf, pb); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	logger.Debugf("json Marshal %T %s", v, string(debug.Stack()))
	return json.Marshal(v)
}

func (j Marshaler) Unmarshal(d []byte, v interface{}) error {
	if pb, ok := v.(proto.Message); ok {
		logger.Debugf("jsonpb Unmarshal %T %s", v, string(debug.Stack()))
		return jsonpb.Unmarshal(bytes.NewReader(d), pb)
	}
	logger.Debugf("json Unmarshal %T %s", v, string(debug.Stack()))
	return json.Unmarshal(d, v)
}

func (j Marshaler) String() string {
	return "json"
}
