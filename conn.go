package algeneva

import (
	"net"
)

// conn is a wrapper around a net.Conn that applies strategies to http requests sent from a client and
// encrypts/decrypts the body of the request and response.
type conn struct {
	net.Conn
	strategies []strategy
}

// Write wraps p using conn.wrapper and if the connection is a client then will apply strategies before writing to
// the connection. It returns the number of bytes written and any error.
func (c *conn) Write(p []byte) (n int, err error) {
	req, err := newRequest(p)
	if err != nil {
		return 0, err
	}

	c.strategies[0].apply(req)
	return c.Conn.Write(req.bytes())
}
