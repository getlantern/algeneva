package algeneva

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAction(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		left    action
		right   action
		want    action
		wantErr bool
	}{
		{
			name:    "error: missing closing arg brace",
			action:  "unknown",
			wantErr: true,
		}, {
			name:    "error: non duplicate action with right action",
			action:  "chagecase{upper}",
			right:   testChangecaseAction(),
			wantErr: true,
		}, {
			name:    "error: changecase missing args",
			action:  "changecase",
			wantErr: true,
		}, {
			name:    "error: insert missing args",
			action:  "insert{a0:a1}",
			wantErr: true,
		}, {
			name:    "error: replace missing args",
			action:  "replace{a0:a1}",
			wantErr: true,
		}, {
			name:    "error: duplicate args",
			action:  "duplicate{arg}",
			wantErr: true,
		}, {
			name:    "error: unknown action",
			action:  "unknown",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newAction(tt.action, tt.left, tt.right)
			testIfErrorOrEqual(t, tt.wantErr, err, tt.want, got)
		})
	}
}

func TestChangeCaseAction_Apply(t *testing.T) {
	tests := []struct {
		name  string
		field field
		want  field
	}{
		{
			name:  "header",
			field: field{name: "header", value: "value", isHeader: true},
			want:  field{name: "HEADER", value: "VALUE", isHeader: true},
		},
		{
			name:  "not header",
			field: field{name: "", value: "value", isHeader: false},
			want:  field{name: "", value: "VALUE", isHeader: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &changecaseAction{
				Case: "upper",
				next: &terminateAction{},
			}

			got := a.apply(tt.field)
			assert.Equal(t, tt.want, got[0])
		})
	}
}

func TestInsertAction_Apply(t *testing.T) {
	type conf struct {
		Value     string
		Location  string
		Component string
		Num       int
	}
	tests := []struct {
		name  string
		conf  conf
		field field
		want  field
	}{
		{
			name:  "insert mid",
			conf:  conf{Value: "[]", Location: "middle", Component: "value", Num: 2},
			field: field{name: "name", value: "value", isHeader: true},
			want:  field{name: "name", value: "va[][]lue", isHeader: true},
		}, {
			name:  "insert random not start or end",
			conf:  conf{Value: "[]", Location: "random", Component: "value", Num: 2},
			field: field{name: "name", value: "vl", isHeader: true},
			want:  field{name: "name", value: "v[][]l", isHeader: true},
		}, {
			name:  "insert ignore component=name if not header",
			conf:  conf{Value: "[]", Location: "start", Component: "name", Num: 2},
			field: field{name: "", value: "vl", isHeader: false},
			want:  field{name: "", value: "[][]vl", isHeader: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := newInsertAction(tt.conf.Value,
				tt.conf.Location,
				tt.conf.Component,
				tt.conf.Num,
				nil,
			)
			assert.NoError(t, err)

			got := a.apply(tt.field)
			assert.Equal(t, tt.want, got[0])
		})
	}
}

func TestReplaceAction_Apply(t *testing.T) {
	type conf struct {
		Value     string
		Component string
		Num       int
	}
	tests := []struct {
		name  string
		conf  conf
		field field
		want  field
	}{
		{
			name:  "replace name",
			conf:  conf{Value: "[]", Component: "name", Num: 2},
			field: field{name: "name", value: "value", isHeader: true},
			want:  field{name: "[][]", value: "value", isHeader: true},
		},
		{
			name:  "replace ignore component=name if not header",
			conf:  conf{Value: "[]", Component: "name", Num: 2},
			field: field{name: "", value: "value", isHeader: false},
			want:  field{name: "", value: "[][]", isHeader: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := newReplaceAction(tt.conf.Value, tt.conf.Component, tt.conf.Num, nil)
			got := a.apply(tt.field)
			assert.Equal(t, tt.want, got[0])
		})
	}
}

func TestDuplicateAction_Apply(t *testing.T) {
	type actions struct {
		LeftAction  action
		RightAction action
	}
	tests := []struct {
		name    string
		actions actions
		field   field
		want    []field
	}{
		{
			name:    "duplicate no actions",
			actions: actions{nil, nil},
			field:   field{name: "name", value: "value"},
			want: []field{
				{name: "name", value: "value"},
				{name: "name", value: "value"},
			},
		}, {
			name: "duplicate 1 action",
			actions: actions{
				nil,
				&changecaseAction{
					Case: "upper",
					next: &terminateAction{},
				},
			},
			field: field{name: "name", value: "value"},
			want: []field{
				{name: "name", value: "value"},
				{name: "NAME", value: "VALUE"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newDuplicateAction(tt.actions.LeftAction, tt.actions.RightAction)
			got := a.apply(tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}
