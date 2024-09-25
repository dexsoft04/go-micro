// Package proto provides a proto codec
package proto

import (
	"go-micro.dev/v5/logger"
	"io"
	"runtime/debug"

	"github.com/golang/protobuf/proto"
	"go-micro.dev/v5/codec"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	logger.Debugf("proto ReadHeader mt:%v m.Type:%v %s", t, m.Type, string(debug.Stack()))

	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	if b == nil {
		return nil
	}
	logger.Debugf("proto ReadBody b:%T %s", b, string(debug.Stack()))

	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}
	m, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}
	return proto.Unmarshal(buf, m)
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	logger.Debugf("proto Write b:%T m.Type:%v %s", b, m.Type, string(debug.Stack()))
	if b == nil {
		// Nothing to write
		return nil
	}
	p, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}
	buf, err := proto.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(buf)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "proto"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
