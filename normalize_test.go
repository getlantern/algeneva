package algeneva

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
