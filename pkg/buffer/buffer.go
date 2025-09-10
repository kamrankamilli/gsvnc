package buffer

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// ReadWriter is a buffer read/writer for RFB connections.
type ReadWriter struct {
	br *bufio.Reader
	bw *bufio.Writer

	wq        chan []byte
	closeOnce sync.Once
	closed    uint32 // 0=open, 1=closed
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
					// writer is dead; mark closed so producers drop immediately
					rw.Close()
					return
				}
			case <-flushTicker.C:
				if err := rw.flush(); err != nil {
					rw.Close()
					return
				}
			}
		}
	}()
	return rw
}

// Close will stop this buffer from processing messages.
func (rw *ReadWriter) Close() {
	rw.closeOnce.Do(func() {
		atomic.StoreUint32(&rw.closed, 1)
		close(rw.wq)
	})
}

func (rw *ReadWriter) IsClosed() bool { return atomic.LoadUint32(&rw.closed) == 1 }

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
func (rw *ReadWriter) Read(v interface{}) error { return binary.Read(rw.br, binary.BigEndian, v) }

// ReadInto reflects on the given struct and populates its fields from the read buffer.
// The struct fields must be in the order they appear on the buffer.
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
func (rw *ReadWriter) write(v interface{}) error { return binary.Write(rw.bw, binary.BigEndian, v) }

// flush will flush the contents of the write buffer.
func (rw *ReadWriter) flush() error { return rw.bw.Flush() }

// Dispatch will push packed message(s) onto the buffer queue, but will
// drop when closed or when the queue is full (to bound memory).
func (rw *ReadWriter) Dispatch(msg []byte) {
	if rw.IsClosed() {
		return
	}
	// Protect against send-after-close races.
	defer func() { _ = recover() }()

	select {
	case rw.wq <- msg:
	default:
		// queue full — drop the frame instead of growing memory
	}
}
