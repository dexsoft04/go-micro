package grpc

import (
	"encoding/binary"
	"fmt"
	"go-micro.dev/v5/logger"
	"io"
	"runtime/debug"
)

var (
	MaxMessageSize = 1024 * 1024 * 4 // 4Mb
	maxInt         = int(^uint(0) >> 1)
)

func decode(r io.Reader) (uint8, []byte, error) {
	header := make([]byte, 5)

	// read the header
	if _, err := r.Read(header); err != nil {
		return uint8(0), nil, err
	}

	// get encoding format e.g compressed
	cf := uint8(header[0])

	// get message length
	length := binary.BigEndian.Uint32(header[1:])

	// no encoding format
	if length == 0 {
		return cf, nil, nil
	}
	logger.Debugf("Decode: length: %d %T", length, r)
	//
	if int64(length) > int64(maxInt) {
		return cf, nil, fmt.Errorf("grpc: received message larger than max length allowed on current machine (%d vs. %d) %s", length, maxInt, string(debug.Stack()))
	}
	if int(length) > MaxMessageSize {
		return cf, nil, fmt.Errorf("grpc: received message larger than max (%d vs. %d) %s", length, MaxMessageSize, string(debug.Stack()))
	}

	msg := make([]byte, int(length))

	if _, err := r.Read(msg); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		logger.Errorf("Failed to read request: %v", err)
		return cf, nil, err
	}

	return cf, msg, nil
}

func encode(cf uint8, buf []byte, w io.Writer) error {
	header := make([]byte, 5)

	// set compression
	header[0] = byte(cf)

	// write length as header
	binary.BigEndian.PutUint32(header[1:], uint32(len(buf)))

	// read the header
	if _, err := w.Write(header); err != nil {
		logger.Errorf("Failed to encode request: %v", err)
		return err
	}

	// write the buffer
	_, err := w.Write(buf)
	if nil != err {
		logger.Errorf("Failed to encode request: %v", err)
	}
	return err
}
