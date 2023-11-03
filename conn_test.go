package algeneva

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_conn_Write(t *testing.T) {
	req := "GET /route HTTP/1.1\r\nHost: localhost\r\n\r\nsome data"
	strategystr := "[HTTP:host:*]-insert{%20:start:name:1}-|"
	want := "GET /route HTTP/1.1\r\n Host: localhost\r\n\r\nsome data"

	strat, err := newStrategy(strategystr)
	assert.NoError(t, err)

	c := &conn{newTestConn(len(want)), []strategy{strat}}

	n, err := c.Write([]byte(req))
	assert.NoError(t, err)
	assert.Equal(t, len(want), n)

	buf := make([]byte, len(want))
	c.Read(buf)

	assert.Equal(t, want, string(buf))
}

type testConn struct {
	buf bytes.Buffer
}

func newTestConn(n int) *testConn {
	return &testConn{
		buf: bytes.Buffer{},
	}
}

func (c *testConn) Read(b []byte) (n int, err error) {
	return c.buf.Read(b)
}

func (c *testConn) Write(b []byte) (n int, err error) {
	return c.buf.Write(b)
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) LocalAddr() net.Addr {
	return nil
}

func (c *testConn) RemoteAddr() net.Addr {
	return nil
}

func (c *testConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *testConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *testConn) SetWriteDeadline(t time.Time) error {
	return nil
}
