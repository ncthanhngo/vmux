package rpc

import (
	"encoding/json"
	"log/slog"
	"net"
	"sync"
)

// conn is one client connection. All writes funnel through sendCh so the
// single writeLoop goroutine owns the socket writer (no concurrent writes).
type conn struct {
	netConn net.Conn
	sendCh  chan []byte
	done    chan struct{}
	once    sync.Once
}

func newConn(nc net.Conn) *conn {
	return &conn{
		netConn: nc,
		sendCh:  make(chan []byte, sendBuffer),
		done:    make(chan struct{}),
	}
}

// send queues a pre-marshalled line. It blocks while the queue is full (to
// apply backpressure) but abandons the send if the connection is closing.
func (c *conn) send(line []byte) {
	select {
	case c.sendCh <- line:
	case <-c.done:
	}
}

// sendValue marshals v and queues it.
func (c *conn) sendValue(log *slog.Logger, v any) {
	line, err := json.Marshal(v)
	if err != nil {
		log.Error("marshal response", "err", err)
		return
	}
	c.send(line)
}

func (c *conn) writeLoop(log *slog.Logger) {
	for {
		select {
		case <-c.done:
			return
		case line := <-c.sendCh:
			// Copy into a fresh frame: a broadcast line is shared across
			// connections, so appending the newline in place would race.
			frame := make([]byte, 0, len(line)+1)
			frame = append(frame, line...)
			frame = append(frame, '\n')
			if _, err := c.netConn.Write(frame); err != nil {
				log.Debug("conn write failed", "err", err)
				c.close()
				return
			}
		}
	}
}

func (c *conn) close() {
	c.once.Do(func() {
		close(c.done)
		c.netConn.Close()
	})
}
