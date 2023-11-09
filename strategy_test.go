package algeneva

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStrategy(t *testing.T) {
	type args struct {
		strategy string
	}
	tests := []struct {
		name    string
		args    args
		want    Strategy
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStrategy(tt.args.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_Apply(t *testing.T) {
	type fields struct {
		Rules []Rule
	}
	type args struct {
		req *Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Strategy{
				Rules: tt.fields.Rules,
			}
			s.Apply(tt.args.req)
		})
	}
}

func TestTrigger_Match(t *testing.T) {
	type fields struct {
		Proto       string
		TargetField string
		MatchStr    string
	}
	type args struct {
		req *Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Field
		want1  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trigger{
				Proto:       tt.fields.Proto,
				TargetField: tt.fields.TargetField,
				MatchStr:    tt.fields.MatchStr,
			}
			got, got1 := tr.Match(tt.args.req)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Trigger.Match() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Trigger.Match() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_parseRule(t *testing.T) {
	type args struct {
		r string
	}
	tests := []struct {
		name    string
		args    args
		want    Rule
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRule(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseTrigger(t *testing.T) {
	type args struct {
		trigger string
	}
	tests := []struct {
		name    string
		args    args
		want    Trigger
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTrigger(tt.args.trigger)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTrigger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTrigger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseAction(t *testing.T) {
	type args struct {
		action string
	}
	tests := []struct {
		name    string
		args    args
		want    Action
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAction(tt.args.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_splitLeftRight(t *testing.T) {
	type args struct {
		action string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := splitLeftRight(tt.args.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitLeftRight() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("splitLeftRight() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("splitLeftRight() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_applyModifications(t *testing.T) {
	testReq := "GET /route HTTP/1.1\r\nHost: localhost\r\n\r\nsome data"

	tests := []struct {
		name  string
		field Field
		mods  []Field
		want  string
	}{
		{
			name: "modify method",
			field: Field{
				Name:     "method",
				Value:    "GET",
				IsHeader: false,
			},
			mods: []Field{
				{
					Name:     "method",
					Value:    "GET--",
					IsHeader: false,
				},
			},
			want: "GET-- /route HTTP/1.1\r\nHost: localhost\r\n\r\nsome data",
		},
		{
			name: "modify header",
			field: Field{
				Name:     "Host",
				Value:    " localhost",
				IsHeader: true,
			},
			mods: []Field{
				{
					Name:     "aaaaa",
					Value:    " localhost",
					IsHeader: true,
				},
				{
					Name:     "Host",
					Value:    " localhost",
					IsHeader: true,
				},
			},
			want: "GET /route HTTP/1.1\r\naaaaa: localhost\r\nHost: localhost\r\n\r\nsome data",
		},
	}
	for _, tt := range tests {
		req, _ := NewRequest([]byte(testReq))
		t.Run(tt.name, func(t *testing.T) {
			applyModifications(req, tt.field, tt.mods)
			assert.Equal(t, tt.want, string(req.Bytes()))
		})
	}
}
