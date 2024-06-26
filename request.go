package algeneva

import (
	"bytes"
	"fmt"
	"strings"
)

// request is an extremely simple HTTP request parser. It only parses the method, path, and version from the start
// line, and separates the headers and body. It does not parse the headers or body.
type request struct {
	method  string
	path    string
	version string
	headers string
	body    []byte
}

// newRequest parses a byte slice, req, into a request. newRequest returns an error if req is not a valid HTTP request.
func newRequest(req []byte) (*request, error) {
	// Find the index of the end of the headers.
	idx := bytes.Index(req, []byte("\r\n\r\n"))
	if idx == -1 {
		return nil, fmt.Errorf("invalid request: %s", req)
	}

	// Split the request into the start line, rest, and body.
	startLine, headers, _ := bytes.Cut(req[:idx], []byte("\r\n"))
	// Split the start line into the method, path, and version.
	mpv := strings.Split(string(startLine), " ")
	if len(mpv) != 3 {
		return nil, fmt.Errorf("invalid request: %s", req)
	}

	// The scope of application layer Geveva was specifically for HTTP version 1 (HTTP/1.0 and HTTP/1.1), so we only
	// support HTTP/1.0 and HTTP/1.1. (page 5)
	if mpv[2] != "HTTP/1.0" && mpv[2] != "HTTP/1.1" {
		return nil, fmt.Errorf("unsupported HTTP version: %s", mpv[2])
	}

	return &request{
		method:  mpv[0],
		path:    mpv[1],
		version: mpv[2],
		headers: string(headers),
		body:    req[idx+4:],
	}, nil
}

// bytes merges the head and body of the request back into a []byte and returns it.
func (r *request) bytes() []byte {
	head := fmt.Sprintf("%s %s %s\r\n%s\r\n\r\n", r.method, r.path, r.version, r.headers)

	size := len(head) + len(r.body)
	buf := make([]byte, size)

	copy(buf, head)
	copy(buf[len(head):], r.body)

	return buf
}

// getHeader returns the full header, including the name, if it exists. getHeader is case insensitive.
func (r *request) getHeader(name string) string {
	headers := strings.ToLower(r.headers)
	idx := strings.Index(headers, name+":")
	if idx == -1 {
		return ""
	}

	nl := strings.Index(r.headers[idx:], "\r\n")
	if nl == -1 {
		nl = len(r.headers[idx:])
	}

	return r.headers[idx : idx+nl]
}
