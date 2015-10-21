// Package splunk sends log data to a splunk server.
package splunk

import (
	"io"
	"net"
)

type splunk struct {
	addr string
	conn net.Conn
}

// New creates a new writer that sends data
// to the given TCP address.
// It connects on the first call to Write,
// and attempts to reconnect when necessary.
func New(addr string) io.Writer {
	return &splunk{addr: addr}
}

func (s *splunk) Write(p []byte) (n int, err error) {
	if s.conn == nil {
		s.conn, err = net.Dial("tcp", s.addr)
		if err != nil {
			return 0, err
		}
	}
	n, err = s.conn.Write(p)
	if t, ok := err.(net.Error); !ok || !t.Temporary() {
		s.conn.Close()
		s.conn = nil
	}
	return n, err
}
