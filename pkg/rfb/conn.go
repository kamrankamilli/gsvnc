package rfb

import (
	"net"
	"runtime"
	"runtime/debug"

	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/events"
)

// Conn represents a client connection.
type Conn struct {
	c       net.Conn
	s       *Server
	buf     *buffer.ReadWriter
	display *display.Display
}

func (s *Server) newConn(c net.Conn) *Conn {
	buf := buffer.NewReadWriteBuffer(c)
	return &Conn{
		c:   c,
		s:   s,
		buf: buf,
		display: display.NewDisplay(&display.Opts{
			Width:           s.width,
			Height:          s.height,
			Buffer:          buf,
			DisplayProvider: s.displayProvider,
			GetEncodingFunc: s.GetEncoding,
		}),
	}
}

func (c *Conn) serve() {
	defer c.c.Close()
	defer c.buf.Close() // stop writer goroutine immediately

	// Ensure freed pages are returned quickly after this connection ends.
	defer func() {
		runtime.GC()
		debug.FreeOSMemory()
	}()

	if err := c.display.Start(); err != nil {
		log.Errorf("Error starting display: %s", err)
		return
	}
	defer c.display.Close()

	eventHandlers := c.s.GetEventHandlerMap()
	defer events.CloseEventHandlers(eventHandlers)

	for {
		cmd, err := c.buf.ReadByte()
		if err != nil {
			log.Errorf("Client disconnect: %s", err.Error())
			return
		}
		if hdlr, ok := eventHandlers[cmd]; ok {
			if err := hdlr.Handle(c.buf, c.display); err != nil {
				log.Errorf("Error handling cmd %d: %s", cmd, err.Error())
				return
			}
		} else {
			log.Warningf("Unsupported command type %d from client\n", int(cmd))
		}
	}
}
