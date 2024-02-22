package algeneva

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			"name: extra whitespace",
			" Host: example.com",
			"Host: example.com",
		}, {
			"invalid host chars",
			"Host: e>xample.com",
			"Host: example.com",
		}, {
			"value: non-printable chars",
			"Content-Type: \x10text/html; charset=utf-8",
			"Content-Type: text/html; charset=utf-8",
		}, {
			"name: invalid chars",
			"C>ontent-Type: text/html; charset=utf-8",
			"Content-Type: text/html; charset=utf-8",
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

func TestNormalizeRequest(t *testing.T) {
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
			got, err := NormalizeRequest([]byte(tt.req))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, string(got))
			}
		})
	}
}
