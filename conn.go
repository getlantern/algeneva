package algeneva

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
)

// conn implements the net.Conn interface and is a wrapper around a net.Conn that applies strategies to http requests
// sent from a client.
type conn struct {
	net.Conn
	// strategy is the strategy to apply to requests sent on the connection.
	strategy strategy
	// buf is a buffer used to store the request until the headers have been parsed.
	buf bytes.Buffer
	// remaining is the number of bytes of the request to be read and sent after finding the end of the headers and
	// applying the strategy (e.i. the body, or what remains of it after sending the buffer).
	remaining uint64
	// readHeaders is a boolean indicating if the headers have been read yet.
	readHeaders bool
}

// Write applies the configured strategy to the request and writes it to the underlying connection.
//
// If the start line and headers have not been read yet, Write will buffer the request until they have. Only after
// they have been read so the strategy can be applied will anything actually be written to the underlying connection.
// Write does not support chunked transfer encoding or upgrading the connection to a WebSocket.
func (c *conn) Write(p []byte) (n int, err error) {
	// TODO: support chunked transfer encoding and upgrading the connection to a WebSocket.

	defer func() {
		// reset the connection state if we encountered an error or if we sent the whole request.
		if err != nil || (c.remaining == 0 && c.readHeaders) {
			c.reset()
		}
	}()

	// since we can't apply the strategy until we've read all the headers, we need to buffer the
	// request until we have.
	if !c.readHeaders {
		c.buf.Write(p)
		idx := bytes.Index(c.buf.Bytes(), []byte("\r\n\r\n"))
		if idx == -1 {
			return len(p), nil
		}

		// now that we have the headers, we can parse the start line and headers.
		req, err := newRequest(c.buf.Bytes())
		if err != nil {
			return 0, err
		}

		// get the content-length header so we know how many bytes of the request are left to read.
		clh := req.getHeader("content-length")
		cls := strings.Split(clh, ":")
		if len(cls) != 2 {
			return 0, errors.New("missing content-length header")
		}

		c.remaining, err = strconv.ParseUint(textproto.TrimString(cls[1]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid content-length header: %w", err)
		}

		c.readHeaders = true

		// apply the strategy to the request and write it to the underlying connection.
		c.strategy.apply(req)
		if _, err = c.Conn.Write(req.bytes()); err != nil {
			return 0, err
		}

		// subtract the length of req.body in case some the request body was included in p.
		c.remaining -= uint64(len(req.body))
		return len(p), nil
	}

	// if we've already read the headers, we can just write p to the underlying connection.
	n, err = c.Conn.Write(p)
	c.remaining -= uint64(n)

	return n, err
}

// reset resets the connection state.
func (c *conn) reset() {
	c.buf.Reset()
	c.remaining = 0
	c.readHeaders = false
}
