// Package text reads any text/* content-type
package text

import (
	"fmt"
	"go-micro.dev/v5/logger"
	"io"
	"runtime/debug"

	"go-micro.dev/v5/codec"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

// Frame gives us the ability to define raw data to send over the pipes.
type Frame struct {
	Data []byte
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	logger.Tracef("text ReadHeader %v %s", t, string(debug.Stack()))

	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	logger.Tracef("text ReadBody %T %s", b, string(debug.Stack()))

	// read bytes
	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	switch v := b.(type) {
	case *string:
		*v = string(buf)
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
	logger.Tracef("text Write Type:%v %s", m.Type, string(debug.Stack()))

	var v []byte
	switch ve := b.(type) {
	case *Frame:
		v = ve.Data
	case *[]byte:
		v = *ve
	case *string:
		v = []byte(*ve)
	case string:
		v = []byte(ve)
	case []byte:
		v = ve
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
	return "text"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
