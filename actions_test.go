package geneva

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangeCaseAction_Apply(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  Field
	}{
		{
			name:  "header",
			field: Field{Name: "header", Value: "value", IsHeader: true},
			want:  Field{Name: "HEADER", Value: "VALUE", IsHeader: true},
		},
		{
			name:  "not header",
			field: Field{Name: "", Value: "value", IsHeader: false},
			want:  Field{Name: "", Value: "VALUE", IsHeader: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ChangecaseAction{
				Case: "upper",
				Next: &TerminateAction{},
			}

			got := a.Apply(tt.field)
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
		field Field
		want  Field
	}{
		{
			name:  "insert mid",
			conf:  conf{Value: "[]", Location: "mid", Component: "value", Num: 2},
			field: Field{Name: "name", Value: "value", IsHeader: true},
			want:  Field{Name: "name", Value: "va[][]lue", IsHeader: true},
		}, {
			name:  "insert random not start or end",
			conf:  conf{Value: "[]", Location: "random", Component: "value", Num: 2},
			field: Field{Name: "name", Value: "vl", IsHeader: true},
			want:  Field{Name: "name", Value: "v[][]l", IsHeader: true},
		}, {
			name:  "insert ignore component=name if not header",
			conf:  conf{Value: "[]", Location: "start", Component: "name", Num: 2},
			field: Field{Name: "", Value: "vl", IsHeader: false},
			want:  Field{Name: "", Value: "[][]vl", IsHeader: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := NewInsertAction(tt.conf.Value,
				tt.conf.Location,
				tt.conf.Component,
				tt.conf.Num,
				nil,
			)

			got := a.Apply(tt.field)
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
		field Field
		want  Field
	}{
		{
			name:  "replace name",
			conf:  conf{Value: "[]", Component: "name", Num: 2},
			field: Field{Name: "name", Value: "value", IsHeader: true},
			want:  Field{Name: "[][]", Value: "value", IsHeader: true},
		},
		{
			name:  "replace ignore component=name if not header",
			conf:  conf{Value: "[]", Component: "name", Num: 2},
			field: Field{Name: "", Value: "value", IsHeader: false},
			want:  Field{Name: "", Value: "[][]", IsHeader: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := NewReplaceAction(tt.conf.Value, tt.conf.Component, tt.conf.Num, nil)
			got := a.Apply(tt.field)
			assert.Equal(t, tt.want, got[0])
		})
	}
}

func TestDuplicateAction_Apply(t *testing.T) {
	type actions struct {
		LeftAction  Action
		RightAction Action
	}
	tests := []struct {
		name    string
		actions actions
		field   Field
		want    []Field
	}{
		{
			name:    "duplicate no actions",
			actions: actions{nil, nil},
			field:   Field{Name: "name", Value: "value"},
			want: []Field{
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
			},
		}, {
			name: "duplicate 1 action",
			actions: actions{
				nil,
				&ChangecaseAction{
					Case: "upper",
					Next: &TerminateAction{},
				},
			},
			field: Field{Name: "name", Value: "value"},
			want: []Field{
				{Name: "name", Value: "value"},
				{Name: "NAME", Value: "VALUE"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewDuplicateAction(tt.actions.LeftAction, tt.actions.RightAction)
			got := a.Apply(tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}
