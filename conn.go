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

// Write applies a strategy to p, encrypts the data, and writes it to the connection. It returns the number of bytes
// written and any error.
func (c *conn) Write(p []byte) (n int, err error) {
	req, err := newRequest(p)
	if err != nil {
		return 0, err
	}

	c.strategies[0].apply(req)
	req.body = encrypt(req.body)

	return c.Conn.Write(req.bytes())
}

// Read reads data from the connection and decrypts it. It returns the number of bytes read and any error encountered.
func (c *conn) Read(p []byte) (n int, err error) {
	buf := make([]byte, len(p))
	n, err = c.Conn.Read(buf)
	if err != nil {
		return 0, err
	}

	req, err := newRequest(buf[:n])
	if err != nil {
		return 0, err
	}

	req.body = decrypt(req.body)
	copy(p, req.bytes())
	return n, nil
}

func (c *conn) Close() error {
	return c.Conn.Close()
}
