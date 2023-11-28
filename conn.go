package algeneva

import (
	"net"
	"time"
)

// Conn is a wrapper around a net.Conn that applies strategies to http requests sent from a client.
type Conn struct {
	conn     net.Conn
	strategy strategy
}

// Write wraps p using conn.wrapper and if the connection is a client then will apply strategies before writing to
// the connection. It returns the number of bytes written and any error.
func (c *Conn) Write(p []byte) (n int, err error) {
	req, err := newRequest(p)
	if err != nil {
		return 0, err
	}

	c.strategy.apply(req)
	return c.conn.Write(req.bytes())
}

// Read reads data from the connection and returns the number of bytes read and any error.
func (c *Conn) Read(p []byte) (n int, err error) {
	return c.conn.Read(p)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (conn *Conn) Close() error {
	return conn.conn.Close()
}

// LocalAddr returns the local network address, if known.
func (conn *Conn) LocalAddr() net.Addr {
	return conn.conn.LocalAddr()
}

// RemoteAddr returns the remote network address, if known.
func (conn *Conn) RemoteAddr() net.Addr {
	return conn.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail instead of blocking. The deadline applies to all future
// and pending I/O, not just the immediately following call to
// Read or Write. After a deadline has been exceeded, the
// connection can be refreshed by setting a deadline in the future.
//
// If the deadline is exceeded a call to Read or Write or to other
// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
// The error's Timeout method will return true, but note that there
// are other possible errors for which the Timeout method will
// return true even if the deadline has not been exceeded.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (conn *Conn) SetDeadline(t time.Time) error {
	return conn.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (conn *Conn) SetReadDeadline(t time.Time) error {
	return conn.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (conn *Conn) SetWriteDeadline(t time.Time) error {
	return conn.conn.SetWriteDeadline(t)
}
