package algeneva

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConn_Write(t *testing.T) {
	req := "GET /route HTTP/1.1\r\nHost: localhost\r\nContent-Length: 9\r\n\r\nsome data"
	want := "GET /route HTTP/1.1\r\n Host: localhost\r\nContent-Length: 9\r\n\r\nsome data"
	tests := []struct {
		name      string
		req       string
		writeSize int
		wantErr   bool
	}{
		{
			name:      "full request",
			req:       req,
			writeSize: len(req),
			wantErr:   false,
		}, {
			name:      "multiple writes, headers first",
			req:       req,
			writeSize: strings.Index(req, "\r\n\r\n") + 4,
			wantErr:   false,
		}, {
			name:      "multiple header writes",
			req:       req,
			writeSize: strings.Index(req, "\r\n\r\n") / 2,
			wantErr:   false,
		}, {
			name:      "multiple writes, partial body",
			req:       req,
			writeSize: strings.Index(req, "\r\n\r\n") + 4 + 4,
			wantErr:   false,
		}, {
			name:      "error: missing content-length header",
			req:       strings.ReplaceAll(req, "Content-Length: 9\r\n", ""),
			writeSize: len(req),
			wantErr:   true,
		},
	}

	strategystr := "[HTTP:host:*]-insert{%20:start:name:1}-|"

	strat, err := newStrategy(strategystr)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conn{
				Conn:     &testConn{},
				strategy: strat,
			}

			var err error
			for i := 0; i < len(tt.req); i += tt.writeSize {
				j := i + tt.writeSize
				if j > len(tt.req) {
					j = len(tt.req)
				}

				if _, err = c.Write([]byte(tt.req[i:j])); err != nil {
					break
				}
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				buf := make([]byte, len(want))
				c.Read(buf)

				assert.Equal(t, want, string(buf))
			}

			assert.True(t, !c.readHeaders && c.remaining == 0)
		})
	}
}

type testConn struct {
	buf bytes.Buffer
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
