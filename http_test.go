package algeneva

import (
	"bufio"
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteRequest(t *testing.T) {
	strategy, _ := NewHTTPStrategy("[HTTP:method:*]-insert{%41:end:value:2}-|")
	req, _ := http.NewRequest("CONNECT", "http://example.com:80", nil)
	want := "CONNECTAA example.com:80 HTTP/1.1\r\nHost: example.com:80\r\n\r\n"
	w := bytes.NewBuffer(make([]byte, 0, 1024))
	req.Header.Set("User-Agent", "")
	err := WriteRequest(w, req, strategy)
	require.NoError(t, err)
	assert.Equal(t, want, w.String())
}

func TestReadRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     string
		want    string
		wantErr bool
	}{
		{
			"no modifications",
			"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			false,
		}, {
			"invalid method, default to GET",
			"GXET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			false,
		}, {
			"invalid version, default to HTTP/1.1",
			"GET  /  version\r\nHost: example.com\r\n\r\n",
			"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			false,
		}, {
			"correct URI with host for CONNECT",
			"CONNECT / HTTP/1.1\r\nHost: www.google.com\r\n\r\n",
			"CONNECT www.google.com:80 HTTP/1.1\r\nHost: www.google.com:80\r\n\r\n",
			false,
		}, {
			"clean header",
			"GET / HTTP/1.1\r\nHost: \r example.com\r\n\r\n",
			"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			false,
		}, {
			"multiple headers",
			"GET / HTTP/1.1\r\nHost: example.com\r\nA: b\r\n\r\n",
			"GET / HTTP/1.1\r\nHost: example.com\r\nA: b\r\n\r\n",
			false,
		}, {
			"missing header body separator",
			"GET / HTTP/1.1\r\nHost: example.com",
			"",
			true,
		}, {
			"missing component",
			"/ HTTP/<1.1\r\nHost: example.com\r\n\r\n",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bufio.NewReader(strings.NewReader(tt.req))
			got, err := ReadRequest(b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				b.Reset(strings.NewReader(tt.want))
				want, _ := http.ReadRequest(b)
				assert.Equal(t, want, got)
			}
		})
	}
}

func TestCleanHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			"cleaned",
			"Host: example.com",
			"Host: example.com",
		}, {
			"extra whitespace in name",
			" Host: example.com",
			"Host: example.com",
		}, {
			"invalid host chars",
			"Host: e>xample.com",
			"Host: example.com",
		}, {
			"non-printable chars in value",
			"Content-Type: \x10text/html; charset=utf-8",
			"Content-Type: text/html; charset=utf-8",
		}, {
			"invalid chars in name",
			"C>ontent-Type: text/html; charset=utf-8",
			"Content-Type: text/html; charset=utf-8",
		}, {
			"clean header",
			"Host: \r example.com",
			"Host: example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanHeader([]byte(tt.header))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

type testReqLine struct{ method, path, version string }

func TestParseRequestLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    testReqLine
		wantErr bool
	}{
		{
			"no modifications",
			"GET / HTTP/1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"absolute URI",
			" GET http://example.com HTTP/1.1",
			testReqLine{"GET", "http://example.com", "HTTP/1.1"},
			false,
		}, {
			"leading whitespace",
			" GET / HTTP/1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"excessive whitespace",
			"GET  /  HTTP/1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"invalid chars",
			"G>ET / HTTP/<1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"duplicate method",
			"GET GET / HTTP/1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"duplicate version",
			"GET / HTTP/1.1 HTTP/1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"invalid method",
			"GETX / HTTP/1.1",
			testReqLine{"", "/", "HTTP/1.1"},
			false,
		}, {
			"invalid version",
			"GET / HTTP/1.1X",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"space in path",
			"GET / home HTTP/1.1",
			testReqLine{"GET", "/", "HTTP/1.1"},
			false,
		}, {
			"invalid: missing component",
			"GET HTTP/1.1",
			testReqLine{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, path, version, err := parseRequestLine([]byte(tt.line))
			got := testReqLine{string(method), string(path), string(version)}
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
