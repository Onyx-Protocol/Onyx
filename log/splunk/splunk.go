// Package splunk sends log data to a splunk server.
package splunk

import (
	"io"
	"net"
	"time"
)

const (
	// DialTimeout limits how long a write will block
	// while dialing the splunk server. Assuming the
	// connection stays open for a long time, this will
	// happen only rarely.
	DialTimeout = 50 * time.Millisecond

	// WriteTimeout limits how long a write will block
	// sending data on the network. It is deliberately
	// very small, so that writes can only be satisfied
	// by the local network buffers. It should never
	// block waiting for a TCP ACK for an appreciable
	// amount of time.
	WriteTimeout = 100 * time.Microsecond
)

type splunk struct {
	addr    string
	conn    net.Conn
	dropmsg []byte
	err     error // last write error
}

// New creates a new writer that sends data
// to the given TCP address.
// It connects on the first call to Write,
// and attempts to reconnect when necessary,
// with a timeout of DialTimeout.
//
// Every write has a timeout of WriteTimeout.
// If the write doesn't complete in that time,
// the writer drops the unwritten data.
// For every contiguous run of dropped data,
// it writes dropmsg before resuming ordinary writes.
// As long as the remote endpoint can keep up
// with the averate data rate and the local
// network buffers in the kernel and NIC are
// big enough to hold traffic bursts, no data
// will be lost.
func New(addr string, dropmsg []byte) io.Writer {
	return &splunk{
		addr:    addr,
		dropmsg: dropmsg,
	}
}

func (s *splunk) Write(p []byte) (n int, err error) {
	if s.conn == nil {
		s.conn, err = net.DialTimeout("tcp", s.addr, DialTimeout)
		if err != nil {
			return 0, err
		}
	}

	if s.err != nil {
		s.conn.SetDeadline(time.Now().Add(WriteTimeout))
		_, s.err = s.conn.Write(s.dropmsg)
	}
	if s.err == nil {
		s.conn.SetDeadline(time.Now().Add(WriteTimeout))
		n, s.err = s.conn.Write(p)
	}

	if t, ok := s.err.(net.Error); s.err != nil && (!ok || !t.Temporary()) {
		s.conn.Close()
		s.conn = nil
	}
	return n, s.err
}
