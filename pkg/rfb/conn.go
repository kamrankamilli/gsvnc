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
	conn := &Conn{
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

	s.connMu.Lock()
	s.connections[conn] = struct{}{}
	s.connMu.Unlock()

	return conn
}

func (c *Conn) serve() {
	defer func() {
		c.c.Close()
		c.buf.Close()
		c.s.removeConn(c) // Remove from tracking
		c.display.Close()

		// Force cleanup
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

func (s *Server) removeConn(conn *Conn) {
	s.connMu.Lock()
	delete(s.connections, conn)
	s.connMu.Unlock()
}

func (s *Server) CloseAllConnections() {
	s.connMu.RLock()
	connections := make([]*Conn, 0, len(s.connections))
	for conn := range s.connections {
		connections = append(connections, conn)
	}
	s.connMu.RUnlock()

	for _, conn := range connections {
		conn.c.Close()
	}
}
