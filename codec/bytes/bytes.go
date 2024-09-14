// Package bytes provides a bytes codec which does not encode or decode anything
package bytes

import (
	"fmt"
	"go-micro.dev/v5/codec"
	"go-micro.dev/v5/logger"
	"io"
	"runtime/debug"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

// Frame gives us the ability to define raw data to send over the pipes.
type Frame struct {
	Data []byte
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	logger.Debugf("ReadHeader m.Type:%v %s", m.Type, string(debug.Stack()))

	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	logger.Debugf("ReadBody m.Type:%T %s", b, string(debug.Stack()))

	// read bytes
	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	switch v := b.(type) {
	case *[]byte:
		*v = buf
	case *Frame:
		v.Data = buf
	default:
		return fmt.Errorf("failed to read body: %v is not type of *[]byte", b)
	}

	return nil
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	logger.Debugf("Write m.Type:%T %s", b, string(debug.Stack()))

	var v []byte
	switch vb := b.(type) {
	case *Frame:
		v = vb.Data
	case *[]byte:
		v = *vb
	case []byte:
		v = vb
	default:
		return fmt.Errorf("failed to write: %v is not type of *[]byte or []byte", b)
	}
	_, err := c.Conn.Write(v)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "bytes"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
