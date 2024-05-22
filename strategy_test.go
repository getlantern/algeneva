package algeneva

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		want     *HTTPStrategy
		wantErr  bool
	}{
		{
			name:     "valid strategy",
			strategy: "[http:path:*]-changecase{upper}-|",
			want: &HTTPStrategy{
				rules: []rule{
					{
						trigger: trigger{proto: "HTTP", targetField: "path", matchStr: "*"},
						tree:    testChangecaseAction(),
					},
				},
			},
			wantErr: false,
		}, {
			name:     "invalid format",
			strategy: "[http:path:*]-changecase{upper}",
			want:     nil,
			wantErr:  true,
		}, {
			name:     "no rules",
			strategy: "",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPStrategy(tt.strategy)
			testIfErrorOrEqual(t, tt.wantErr, err, tt.want, got)
		})
	}
}

func Test_parseRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    string
		want    rule
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRule(tt.rule)
			testIfErrorOrEqual(t, tt.wantErr, err, tt.want, got)
		})
	}
}

func Test_parseTrigger(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		want    trigger
		wantErr bool
	}{
		{
			name:    "valid trigger",
			trigger: "[http:path:*]",
			want: trigger{
				proto:       "HTTP",
				targetField: "path",
				matchStr:    "*",
			},
			wantErr: false,
		}, {
			name:    "error: invalid format",
			trigger: "[http:path:*",
			want:    trigger{},
			wantErr: true,
		}, {
			name:    "error: invalid format, missing argument",
			trigger: "[http:*]",
			want:    trigger{},
			wantErr: true,
		}, {
			name:    "error: unsupported proto",
			trigger: "[icmp:path:*]",
			want:    trigger{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTrigger(tt.trigger)
			testIfErrorOrEqual(t, tt.wantErr, err, tt.want, got)
		})
	}
}

func Test_parseAction(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		want    action
		wantErr bool
	}{
		{
			name:    "terminateIfEmpty",
			action:  "",
			want:    action(&terminateAction{}),
			wantErr: false,
		}, {
			name:    "no subsequent actions",
			action:  "changecase{upper}",
			want:    testChangecaseAction(),
			wantErr: false,
		}, {
			name:   "subsequent actions",
			action: "duplicate(changecase{upper},changecase{upper})",
			want: action(
				&duplicateAction{
					leftAction:  testChangecaseAction(),
					rightAction: testChangecaseAction(),
				},
			),
			wantErr: false,
		}, {
			name:    "error: invalid format missing closing paren",
			action:  "changecase{upper}(,",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAction(tt.action)
			testIfErrorOrEqual(t, tt.wantErr, err, tt.want, got)
		})
	}
}

func Test_splitLeftRight(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		wantLeft  string
		wantRight string
		wantErr   bool
	}{
		{
			name:      "no right",
			action:    "(left,)",
			wantLeft:  "left",
			wantRight: "",
			wantErr:   false,
		}, {
			name:      "no left",
			action:    "(,right)",
			wantLeft:  "",
			wantRight: "right",
			wantErr:   false,
		}, {
			name:      "nested actions",
			action:    "(left(subleft0,subleft1(subsubleft0,)),right(subright0,subright1))",
			wantLeft:  "left(subleft0,subleft1(subsubleft0,))",
			wantRight: "right(subright0,subright1)",
			wantErr:   false,
		}, {
			name:      "error: invalid format",
			action:    "(left)",
			wantLeft:  "",
			wantRight: "",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLeft, gotRight, err := splitLeftRight(tt.action)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantLeft, gotLeft)
				assert.Equal(t, tt.wantRight, gotRight)
			}
		})
	}
}

func Test_applyModifications(t *testing.T) {
	tests := []struct {
		name  string
		field field
		mods  []field
		want  string
	}{
		{
			name: "modify method",
			field: field{
				name:     "method",
				value:    "GET",
				isHeader: false,
			},
			mods: []field{
				{
					name:     "method",
					value:    "GET--",
					isHeader: false,
				},
			},
			want: "GET-- /route HTTP/1.1\r\nHost: localhost\r\n\r\nsome data",
		},
		{
			name: "modify header",
			field: field{
				name:     "Host",
				value:    " localhost",
				isHeader: true,
			},
			mods: []field{
				{
					name:     "aaaaa",
					value:    " localhost",
					isHeader: true,
				},
				{
					name:     "Host",
					value:    " localhost",
					isHeader: true,
				},
			},
			want: "GET /route HTTP/1.1\r\naaaaa: localhost\r\nHost: localhost\r\n\r\nsome data",
		},
	}
	for _, tt := range tests {
		req := testReq()
		t.Run(tt.name, func(t *testing.T) {
			applyModifications(&req, tt.field, tt.mods)
			assert.Equal(t, tt.want, string(req.bytes()))
		})
	}
}

func testReq() request {
	return request{
		method:  "GET",
		path:    "/route",
		version: "HTTP/1.1",
		headers: "Host: localhost",
		body:    []byte("some data"),
	}
}

func testChangecaseAction() action {
	return action(&changecaseAction{toCase: "upper", next: action(&terminateAction{})})
}

func testIfErrorOrEqual(t *testing.T, wantErr bool, err error, want interface{}, got interface{}) {
	if wantErr {
		assert.Error(t, err)
	} else {
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}
