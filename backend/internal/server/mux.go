// Package server provides shared utilities for the HTTP/HTTPS API server.
package server

import (
	"io"
	"net"
	"sync"
)

// peekedConn is a net.Conn that prepends already-read bytes back to subsequent reads.
type peekedConn struct {
	net.Conn
	buf []byte
}

func (c *peekedConn) Read(b []byte) (int, error) {
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}

// protoListener is a net.Listener backed by a channel of pre-accepted conns.
// Calling Close signals Accept to stop; it does NOT close the underlying TCP listener.
type protoListener struct {
	addr net.Addr
	ch   <-chan net.Conn
	once sync.Once
	done chan struct{}
}

func (l *protoListener) Accept() (net.Conn, error) {
	select {
	case conn, ok := <-l.ch:
		if !ok {
			return nil, net.ErrClosed
		}
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *protoListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *protoListener) Addr() net.Addr { return l.addr }

// DemuxListener splits a single TCP listener into TLS and plain-HTTP sub-listeners
// by peeking at the first byte of each accepted connection.
// Connections starting with 0x16 (TLS Handshake record type) are routed to tlsL;
// all other connections are routed to httpL.
//
// Closing the returned listeners stops them from accepting new connections (and
// drops any in-flight connections whose first byte has not yet been read).
// To stop the demuxer goroutine itself, close the underlying ln directly.
func DemuxListener(ln net.Listener) (tlsL, httpL net.Listener) {
	tlsCh := make(chan net.Conn, 64)
	httpCh := make(chan net.Conn, 64)

	tl := &protoListener{addr: ln.Addr(), ch: tlsCh, done: make(chan struct{})}
	hl := &protoListener{addr: ln.Addr(), ch: httpCh, done: make(chan struct{})}

	var wg sync.WaitGroup
	go func() {
		defer func() {
			wg.Wait()
			closeDraining(tlsCh)
			closeDraining(httpCh)
		}()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				one := make([]byte, 1)
				if _, err := io.ReadFull(c, one); err != nil {
					_ = c.Close()
					return
				}
				pc := &peekedConn{Conn: c, buf: one}
				if one[0] == 0x16 { // TLS Handshake record type
					select {
					case tlsCh <- pc:
					case <-tl.done:
						_ = c.Close()
					}
				} else {
					select {
					case httpCh <- pc:
					case <-hl.done:
						_ = c.Close()
					}
				}
			}(conn)
		}
	}()

	return tl, hl
}

// closeDraining closes ch and closes any buffered connections that were not consumed.
func closeDraining(ch chan net.Conn) {
	close(ch)
	for c := range ch {
		_ = c.Close()
	}
}
