package buffer

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net"
	"reflect"
	"time"
)

// ReadWriter is a buffer read/writer for RFB connections.
type ReadWriter struct {
	br *bufio.Reader
	bw *bufio.Writer

	wq chan []byte
}

// NewReadWriteBuffer returns a new ReadWriter for the given connection.
func NewReadWriteBuffer(c net.Conn) *ReadWriter {
	rw := &ReadWriter{
		br: bufio.NewReader(c),
		bw: bufio.NewWriterSize(c, 256<<10),
		wq: make(chan []byte, 100),
	}
	go func() {
		flushTicker := time.NewTicker(5 * time.Millisecond)
		defer flushTicker.Stop()
		for {
			select {
			case msg, ok := <-rw.wq:
				if !ok {
					_ = rw.flush()
					return
				}
				if err := rw.write(msg); err != nil {
					return
				}
			case <-flushTicker.C:
				if err := rw.flush(); err != nil {
					return
				}
			}
		}
	}()
	return rw
}

// Close will stop this buffer from processing messages.
func (rw *ReadWriter) Close() { close(rw.wq) }

// Reader returns a direct reference to the underlying reader.
func (rw *ReadWriter) Reader() *bufio.Reader { return rw.br }

// ReadByte reads a single byte from the buffer.
func (rw *ReadWriter) ReadByte() (byte, error) {
	b, err := rw.br.ReadByte()
	if err != nil {
		return 0, err
	}
	return b, nil
}

// ReadPadding pops padding off the read buffer of the given size.
func (rw *ReadWriter) ReadPadding(size int) error {
	for i := 0; i < size; i++ {
		if _, err := rw.ReadByte(); err != nil {
			return err
		}
	}
	return nil
}

// Read will read from the buffer into the given interface. Not for structs; use ReadInto.
func (rw *ReadWriter) Read(v interface{}) error {
	return binary.Read(rw.br, binary.BigEndian, v)
}

// ReadInto reflects on the given struct and populates its fields from the read buffer.
func (rw *ReadWriter) ReadInto(data interface{}) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("Data is invalid (nil or non-pointer)")
	}
	val := rv.Elem()
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		if err := rw.Read(f.Addr().Interface()); err != nil {
			return err
		}
	}
	return nil
}

// write writes the given interface to the buffer.
func (rw *ReadWriter) write(v interface{}) error {
	return binary.Write(rw.bw, binary.BigEndian, v)
}

// flush will flush the contents of the write buffer.
func (rw *ReadWriter) flush() error { return rw.bw.Flush() }

// Dispatch will push packed message(s) onto the buffer queue.
func (rw *ReadWriter) Dispatch(msg []byte) { rw.wq <- msg }
