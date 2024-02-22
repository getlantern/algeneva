package algeneva

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
)

/*
   RFC spec can be found at https://datatracker.ietf.org/doc/html/rfc7230
   Syntax notation used in this file:
       OWS    = optional whitespace
       SP     = space
       HTAB   = horizontal tab
       CR     = carriage return
       LF     = line feed
       CRLF   = CR LF

       ALPHA  = A-Z / a-z
       DIGIT  = 0-9
       VCHAR  = any visible ASCII character

   Note: There are additional syntax notations defined in the RFC spec, but they are not used in
	this file.
*/

// NormalizeRequest normalizes an HTTP request that was modified with Application-Layer Geneva
// strategies. NormalizeRequest does not reverse Geneva strategies, it only normalizes the request
// to adhere to the HTTP/1.0 and HTTP/1.1 RFCs. Most strategies will be undone by this, but some
// cannot, as it would require knowing the original value that was modified. NormalizeRequest
// does not guarantee that values, such as URI and host, are correct, only that they are valid
// according to the RFCs.
//
// If a valid method or version cannot be found, then the method will default to GET or POST,
// depending on if there is a body or not, and the version will default to HTTP/1.1.
func NormalizeRequest(req []byte) ([]byte, error) {
	// Separate headers and body. The headers must end with "\r\n\r\n", even if body is empty.
	idx := bytes.Index(req, []byte("\r\n\r\n"))
	if idx == -1 {
		return nil, errors.New("missing header/body separator")
	}

	// We keep the first "\r\n" because it is part of the header and so the scanner sees it as a
	// line and not EOF. scanner.Scan will return false if it sees EOF.
	head, body := req[:idx+2], req[idx+4:]

	// parse and normalize request line and headers
	scanner := bufio.NewScanner(bytes.NewReader(head))
	scanner.Split(scanCRLF)
	if !scanner.Scan() {
		return nil, scanner.Err()
	}

	// Even if parseRequestLine successfully parses the request line and err is nil, method and
	// version could still be empty if they were not found.
	method, path, version, err := parseRequestLine(scanner.Bytes())
	if err != nil {
		return nil, err
	}

	// We need to check if method was found. Some strategies modify the method, making it invalid;
	// such as inserting valid charaters or replacing the method entirely.
	//
	// There are three ways to handle an invalid method:
	//    1. Spell check the method and replace it with the correct one. This only works if valid
	//       characters were inserted.
	//    2. Use a default: if there is a body then use POST, otherwise use GET.
	//    3. Return an error. This is not ideal because it will invalidate all Geneva strategies
	//       that modifies the method, even though these work with others servers (e.g. Apache and
	//		   Nginx).
	//
	// For now, we will use the second strategy since it is easier to implement.
	if method == "" {
		if len(body) > 0 {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	// We also need to check version for the same reason. Since Geneva only supports HTTP/1.0 and
	// HTTP/1.1, we will use HTTP/1.1 as the default.
	if version == "" {
		version = "HTTP/1.1"
	}

	// Now clean the headers. We're only going to clean the headers, we'll leave validating them to
	// the caller.
	var headers [][]byte
	hostFnd := false
	for scanner.Scan() {
		h := scanner.Bytes()
		h = append([]byte{}, h...) // Make a copy of h so scanner.Scan doesn't overwrite it.

		h, err := cleanHeader(h)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, h)
		}

		// Since there can only be one host header, we need to check if it was already found. We
		// keep the first one we find and ignore the rest. Maybe we want to keep the last one
		// instead?
		if bytes.HasPrefix(h, []byte("Host:")) {
			if hostFnd {
				continue
			}

			hostFnd = true
		}

		headers = append(headers, h)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Now we need to rebuild the request. req might not be big enough to hold the new request, so
	// we need to create a new buffer.
	rl := []byte(method + " " + path + " " + version)
	headers = append([][]byte{rl}, headers...)
	newHead := bytes.Join(headers, []byte("\r\n"))

	newReq := make([]byte, len(newHead)+4+len(body))

	copy(newReq, []byte(newHead))
	copy(newReq[len(newHead):], []byte("\r\n\r\n"))
	copy(newReq[len(newHead)+4:], body)
	return newReq, nil
}

// parseRequestLine tries to parse and normalize an HTTP request line. parseRequestLine adheres
// loosely to the RFC spec for HTTP/1.0 and HTTP/1.1. If no valid method or version is found, then
// the empty string is returned. An error is returned if there are less than three components after
// removing excess whitespace.
func parseRequestLine(line []byte) (method, path, version string, err error) {
	// We need to parse out each component, which is separated by at least one SP and zero or more
	// OWS. (The spec is more strict than this now, but some servers are not which is why Geneva
	// supports it.)
	//
	// RFC 7230, section 3.1.1.
	//
	//    request line = OWS* method OWS* SP OWS* path OWS* SP OWS* version OWS*
	//             OWS = *( SP / HTAB ) ; Geneva also includes CR
	//
	// We'll also need to clean each component; and since components could be duplicated with
	// modifications or whitespace inserted in the middle, there could be more than 3 (which we'll
	// have to try to filter out later).
	// One way to do this:
	//    | trim leading OWS
	//    | parse upto first SP
	//    | remove trailing OWS
	//    | repeat until there are no more SPs
	//
	//    | finally find and clean each component

	var components [][]byte
	for len(line) > 0 {
		line = bytes.TrimSpace(line)
		sp := bytes.IndexByte(line, ' ')
		if sp == -1 {
			sp = len(line)
		}

		comp := bytes.TrimSpace(line[:sp])
		if len(comp) > 0 {
			components = append(components, comp)
		}

		line = line[sp:]
	}

	if len(components) < 3 {
		return "", "", "", fmt.Errorf("request line has less than 3 components: %q", line)
	}

	// If we have 3 or more components, then we need to clean each component and, if more than 3,
	// try to figure out which component is which. The easiest way to do this is to find the method
	// and version first, as the path must be between them.

	var mIdx, vIdx int

	// Attempt to find method
	for ; mIdx < len(components)-2; mIdx++ {
		c := clean(components[mIdx], isAlpha)
		m := string(c)
		if isValidMethod(m) {
			method = m
			break
		}
	}

	if method == "" {
		// We didn't find a valid method so we reset mIdx.
		mIdx = 0
	}

	// Attempt to find version
	for vIdx = len(components) - 1; vIdx >= 2; vIdx-- {
		c := clean(components[vIdx], func(b byte) bool { return isValidToken(b, versionTokens) })
		v := string(c)
		if isVersion1x(v) {
			version = v
			break
		}
	}

	if version == "" {
		// We didn't find a valid version so reset vIdx.
		vIdx = len(components) - 1
	}

	// The path must be between the method and version. findPath will also check if valid
	// characters were inserted in front of path if in the origin or absolute form or inserted at
	// the front or end of the path if in the asterisk form.
	path = findPath(components[mIdx+1 : vIdx])

	if path == "" {
		// We still didn't find a valid path so it must have been overridden by the replace action.
		// There's no way to know what the original path was so we'll set it to the root.
		path = "/"
	}

	return method, path, version, nil
}

func findPath(components [][]byte) (path string) {
	cleanedComps := make([][]byte, 0, len(components))
	for _, comp := range components {
		comp = clean(comp, func(b byte) bool {
			return isValidToken(b, validTokenTable) || b == '/' || b == ':'
		})
		if isValidPath(comp) {
			// comp matches the origin, absolute, or asterisk form so we assume it's the path and
			// return it.
			return string(comp)
		}

		// We'll keep the cleaned component in case we don't find a valid path.
		cleanedComps = append(cleanedComps, comp)
	}

	// We didn't find a valid path so either it was modified or it was invalid to begin with.
	// Assuming it was modified and since isValidPath reports if it matches the origin, absolutem
	// or asterisk form, we can check if characters were inserted at the beginning and remove them.
	for _, comp := range cleanedComps {
		// Check for the first instance of 'http(s)://' or '/' and return the string from that
		// index to the end, or '*' and return it without the leading or trailing characters.

		// Since it's possible for valid form characters to be inserted anywhere, we can't know
		// where the correct path actually starts if multiple forms are present, e.g. '/*', so we
		// just return the first one we find.

		// First check for 'http(s)://' since '/' is a valid path form.
		// BUG: There is an edge case where 'http(s)://' or 'http(s)://<url w/o top-level domain>
		// is inserted at the beginning of the path and the path is not an absolute form. This will
		// cause an invalid path to be returned.
		comp = bytes.ToLower(comp)
		idx := bytes.Index(comp, []byte("http"))
		if idx != -1 {
			j := 4
			if comp[idx+4] == 's' {
				j++
			}

			if bytes.HasPrefix(comp[idx+j:], []byte("://")) {
				return string(comp[idx:])
			}
		}

		// Now check for '/'
		idx = bytes.IndexByte(comp, '/')
		if idx != -1 {
			return string(comp[idx:])
		}

		// Since '*' is the least common form, we'll check for it last.
		if bytes.IndexByte(comp, '*') != -1 {
			return "*"
		}
	}

	return ""
}

// isValidMethod returns true if method is a valid HTTP method.
func isValidMethod(method string) bool {
	// RFC 7231, section 4.1
	//    method    = "GET"          ; section 4.3.1
	//              | "HEAD"         ; section 4.3.2
	//              | "POST"         ; section 4.3.3
	//              | "PUT"          ; section 4.3.4
	//              | "DELETE"       ; section 4.3.5
	//              | "CONNECT"      ; section 4.3.6
	//              | "OPTIONS"      ; section 4.3.7
	//              | "TRACE"        ; section 4.3.8

	switch method {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE":
		return true
	}

	return false
}

// isValidPath returns true if p is a valid HTTP request path. isValidPath does not check for
// authority-form.
func isValidPath(p []byte) bool {
	// RFC 7230, section 5.3
	//    request-target = origin-form        ; Section 5.3.1
	//                   / absolute-form      ; Section 5.3.2
	//                   / authority-form     ; Section 5.3.3
	//                   / asterisk-form      ; Section 5.3.4
	//
	// We're not to check for authority-form, as that can get complicated. Maybe in the future we
	// can add support for it.

	switch {
	case p[0] == '/': // origin-form
		return true
	case len(p) > 8 && (p[0] == 'H' || p[0] == 'h'): // absolute-form
		p0 := bytes.ToLower(p[:8])
		return bytes.Equal(p0, []byte("http://")) || bytes.Equal(p0, []byte("https://"))
	}

	return bytes.Equal(p, []byte("*")) // asterisk-form
}

// isVersion1x returns true if version is HTTP/1.0 or HTTP/1.1.
func isVersion1x(v string) bool {
	switch v {
	case "HTTP/1.0", "HTTP/1.1", "http/1.0", "http/1.1":
		return true
	}

	return false
}

// clean returns s with all invalid characters removed. clean uses validTokenFn to determine if a
// character is valid.
func clean(s []byte, validTokenFn func(b byte) bool) []byte {
	var cleaned []byte
	for _, b := range s {
		if validTokenFn(b) {
			cleaned = append(cleaned, b)
		}
	}

	return cleaned
}

// validTokenTable is a table of valid tokens for method and header names.
//
// Note that obs-fold (line folding) is not supported, even though it is still currently in the
// spec, as it is obsolete.
var validTokenTable = [127]bool{
	// RFC 7230, section 3.2
	//    header-field   = field-name ":" OWS field-value OWS
	//    field-name     = token
	//
	//    obs-fold       = CRLF 1*( SP / HTAB )
	//                   ; obsolete line folding
	//
	// Section 3.2.6
	//    token          = 1*tchar
	//
	//    tchar          = "!" / "#" / "$" / "%" / "&" / "'" / "*"
	//                   / "+" / "-" / "." / "^" / "_" / "`" / "|" / "~"
	//                   / DIGIT / ALPHA
	//
	// This lets us efficiently check for valid header characters. Plus, it's easier to read than
	// comparing ascii values with <, >, ==.

	'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true,
	'8': true, '9': true,

	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true,
	'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true,
	'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'W': true, 'V': true, 'X': true,
	'Y': true, 'Z': true,

	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true,
	'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true,
	'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true,
	'y': true, 'z': true,

	'!': true, '#': true, '$': true, '%': true, '&': true, '\'': true, '*': true, '+': true,
	'-': true, '.': true, '|': true, '~': true, '^': true, '_': true, '`': true,
}

// isValidToken returns true if b is a valid token. tokenTable is a table of ASCII values.
func isValidToken(b byte, tokenTable [127]bool) bool {
	return b < 127 && tokenTable[b]
}

func isCtrl(b byte) bool {
	return b < ' ' || b == 0x7f // DEL
}

func isAlpha(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

var versionTokens = [127]bool{
	'H': true, 'T': true, 'P': true, 'h': true, 't': true, 'p': true,
	'/': true, '1': true, '.': true, '0': true,
}

// hostTokenTable is a table of valid tokens for host header.
var hostTokenTable = [127]bool{
	// RFC 3986, section 3.2.2
	//
	// This lets us efficiently check for valid host characters. Plus, it's easier to read than
	// comparing ascii values with <, >, ==.
	// Some characters that are valid in other header values are not valid in the host header value,
	// which is why we have a separate table.

	'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true,
	'8': true, '9': true,

	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true,
	'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true,
	'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true,
	'Y': true, 'Z': true,

	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true,
	'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true,
	'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true,
	'y': true, 'z': true,

	// host port delim
	':': true,

	// sub-delims
	'!': true, '$': true, '&': true, '\'': true, '(': true, ')': true, '*': true, '+': true,
	',': true, ';': true, '=': true,

	// unreserved
	'-': true, '.': true, '_': true, '~': true,
}

// validHeaderValueToken returns true if b is a valid header value token.
func validHeaderValueToken(b byte) bool {
	// RFC 7230, section 3.2
	//    header-field   = field-name ":" OWS field-value OWS
	//
	//    field-value    = *( field-content / obs-fold )
	//    field-content  = field-vchar [ 1*( SP / HTAB ) field-vchar ]
	//    field-vchar    = VCHAR / obs-text
	//
	//    VCHAR          = %x21-7E ; visible (printing) characters

	return !isCtrl(b) || b == '\t'
}

// cleanHeader returns h with all invalid characters removed.
//
// Note that obs-fold (line folding) is not supported, even though it is still currently in the
// spec, as it is obsolete.
func cleanHeader(h []byte) ([]byte, error) {
	// RFC 7230, section 3.2
	//    header-field = field-name ":" OWS field-value OWS
	//    field-name   = token
	//
	//    obs-fold     = CRLF 1*( SP / HTAB )
	//                 ; obsolete line folding
	//
	//    field-value  = *( field-content / obs-fold )
	//    field-conten = field-vchar [ 1*( SP / HTAB ) field-vchar ]
	//    field-vchar  = VCHAR / obs-text
	//
	//    VCHAR        = %x21-7E ; visible (printing) characters

	name, value, fnd := bytes.Cut(h, []byte(":"))
	if !fnd {
		return nil, fmt.Errorf("invalid header: %q", h)
	}

	// With the exception of the host header, we can clean both the name and value with the
	// validTokenTable (RFC 7230, section 3.2). The host header value has a different set of valid
	// characters (RFC 3986, section 3.2.2) so we'll use hostTokenTable for that.
	name = clean(name, func(b byte) bool { return isValidToken(b, validTokenTable) })
	hasSepOSP := value[0] == ' '
	if hasSepOSP {
		value = value[1:]
	}

	cname := textproto.CanonicalMIMEHeaderKey(string(name))
	if cname == "Host" {
		value = clean(value, func(b byte) bool { return isValidToken(b, hostTokenTable) })
	} else {
		value = bytes.TrimSpace(value)
		value = clean(value, validHeaderValueToken)
	}

	// Since we only removed characters, we can reuse h so we don't have to allocate a new slice.
	n := copy(h, []byte(cname))
	h[n] = ':'
	n += 1
	if hasSepOSP {
		h[n] = ' '
		n += 1
	}

	n += copy(h[n:], value)
	return h[:n], nil
}

// cleanHeaderValue returns s with all invalid header value characters removed.
func cleanHeaderValue(s []byte) []byte {
	// RFC 7230, section 3.2
	//    header-field   = field-name ":" OWS field-value OWS
	//
	//    field-value    = *( field-content / obs-fold )
	//    field-content  = field-vchar [ 1*( SP / HTAB ) field-vchar ]
	//    field-vchar    = VCHAR / obs-text
	//
	//    VCHAR          = %x21-7E ; visible (printing) characters

	s = bytes.TrimSpace(s)
	i := 0
	for _, b := range s {
		if !isCtrl(b) || b == '\t' {
			s[i] = b
			i++
		}
	}

	return s[:i]
}

// scanCRLF is a split function for a Scanner that returns each line of text ending in a CRLF.
func scanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, data[0:i], nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

// NormalizationTestResults is the results of TestStrategyNormalization.
type NormalizationTestResults struct {
	// Name is the name of the test.
	Name string
	// Request is the original request before applying the strategy and normalization.
	Request string
	// Normalized is the normalized request after applying the strategy and normalization.
	Normalized string
	// Msg describes why the test failed if it did. If the test passed but the normalized request
	// is not the same as the original request, then Msg will describe which elements are different.
	// If the test passed and there are no differences, then Msg will be empty.
	Msg string
	// Pass reports whether the test passed.
	Pass bool
}

// TestStrategyNormalization tests if strategy is a valid strategy and whether a request
// transformed by strategy can be normalized to RFC spec. TestStrategyNormalization applies
// strategy to a set of requests and then tries to normalize them. If successful,
// TestStrategyNormalization will check if the the original request was fully restored during
// normalization or if values were inferred. TestStrategyNormalization returns the results of each
// test and whether the strategy passed all tests.
func TestStrategyNormalization(strategy string) ([]NormalizationTestResults, bool, error) {
	strat, err := NewHTTPStrategy(strategy)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create strategy from %s: %w", strategy, err)
	}

	tests := []NormalizationTestResults{
		{
			Name:    "GET",
			Request: "GET /some/path HTTP/1.1\r\nHost: example.com\r\n\r\n",
		}, {
			Name:    "POST without body",
			Request: "POST /some/path HTTP/1.1\r\nHost: example.com\r\n\r\n",
		}, {
			Name:    "POST with body",
			Request: "POST /some/path HTTP/1.1\r\nHost: example.com\r\n\r\nsome body",
		}, {
			Name:    "PUT with body",
			Request: "PUT /some/path HTTP/1.1\r\nHost: example.com\r\n\r\nsome body",
		},
	}
	for t := 0; t < len(tests); t++ {
		test := &tests[t]
		modReq, err := strat.Apply([]byte(test.Request))
		if err != nil {
			test.Msg = fmt.Sprintf("Failed to apply strategy: %s", err)
			continue
		}

		got, err := NormalizeRequest(modReq)
		test.Normalized = string(got)
		if err != nil {
			test.Msg = fmt.Sprintf("Failed to normalize strategy: %s", err)
			continue
		}

		// We need to check if the normalized request is valid per spec. We can just use
		// http.ReadRequest since it'll do all the checks for us.
		b := bufio.NewReader(bytes.NewReader(got))
		_, err = http.ReadRequest(b)
		if err != nil {
			test.Msg = fmt.Sprintf("Failed to create a http.Request from normalized request: %s", err)
			continue
		}

		test.Pass = true

		// At this point, we can guarantee that the normalized request is valid. However, the
		// normalized request might not be the same as the original, so we check if the original
		// request was fully restored during normalization. If not, then we report which elements
		// were not restored. This is not a failure, but it is useful for the user to know.
		diffs, _ := getNormalizeTestDiff([]byte(test.Request), got)
		if len(diffs) > 0 {
			test.Msg = fmt.Sprintf(
				"Could not fully restore original request during normalization. %v",
				strings.Join(diffs, ", "),
			)
		}
	}

	// Check whether the test as a whole passed. If any test failed, then the whole test failed.
	passed := true
	for _, test := range tests {
		passed = passed && test.Pass
	}

	return tests, passed, nil
}

// getNormalizeTestDiff compares the original request with the normalized request and reports any
// differences. getNormalizeTestDiff only compares the method, path, version, and host.
func getNormalizeTestDiff(orig, norm []byte) ([]string, error) {
	// create a request from the original request
	oReq, err := newRequest(orig)
	if err != nil {
		return nil, fmt.Errorf("orig: %w", err)
	}

	// create a request from the normalized request
	nReq, err := newRequest(norm)
	if err != nil {
		return nil, fmt.Errorf("norm: %w", err)
	}

	// We only need to compare the method, path, version, and host. We don't need to compare the
	// any other headers since host is a header itself and the logic to normalize it is the same.
	// Also, currently, host is the only header that Geneva modifies.
	var elemDiffs []string
	if oReq.method != nReq.method {
		elemDiffs = append(elemDiffs, fmt.Sprintf("method: orig=%s, norm=%s", oReq.method, nReq.method))
	}

	if oReq.path != nReq.path {
		elemDiffs = append(elemDiffs, fmt.Sprintf("path: orig=%s, norm=%s", oReq.path, nReq.path))
	}

	if oReq.version != nReq.version {
		elemDiffs = append(elemDiffs, fmt.Sprintf("version: orig=%s, norm=%s", oReq.version, nReq.version))
	}

	getHostForComp := func(req *request) string {
		h := req.getHeader("host")
		h = strings.ToLower(h)
		return strings.TrimSpace(strings.TrimPrefix(h, "host:"))
	}

	oHost := getHostForComp(oReq)
	nHost := getHostForComp(nReq)
	if oHost != nHost {
		elemDiffs = append(elemDiffs, fmt.Sprintf("host: orig=%s, norm=%s", oHost, nHost))
	}

	return elemDiffs, nil
}
