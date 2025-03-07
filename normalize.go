package algeneva

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

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
	r, err := ReadRequest(bufio.NewReader(bytes.NewReader(req)))
	if err != nil {
		return nil, err
	}

	// set user-agent to empty string if it doesn't exist to avoid go adding a default value
	if _, ok := r.Header["User-Agent"]; !ok {
		r.Header.Set("User-Agent", "")
	}
	var b bytes.Buffer
	r.Write(&b)
	return b.Bytes(), nil
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
