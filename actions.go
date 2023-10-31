package geneva

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// Action is an interface that all actions must implement.
type Action interface {
	// String returns a string representation of the action.
	String() string
	// Apply applies the action to the field and returns the result of the action.
	Apply(field Field) []Field
}

func NewAction(action string, next Action) (Action, error) {
	br := strings.Index(action, "{")
	var args []string
	if br != -1 {
		args = strings.Split(action[br:], ":")
		action = action[:br]
	}

	switch action {
	case "changecase":
		if len(args) != 1 {
			return nil, fmt.Errorf("changecase requires 1 argument")
		}

		return NewChangecaseAction(args[0], next)
	case "insert":
		if len(args) != 4 {
			return nil, fmt.Errorf("insert requires 4 arguments")
		}

		n, _ := strconv.Atoi(args[3])
		return NewInsertAction(args[0], args[1], args[2], n, next)
	case "replace":
		if len(args) != 3 {
			return nil, fmt.Errorf("insert requires 3 arguments")
		}

		n, _ := strconv.Atoi(args[2])
		return NewReplaceAction(args[0], args[1], n, next)
	case "duplicate":
		return NewDuplicateAction(next, nil), nil
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}
}

// Field is the target field to apply an action to.
type Field struct {
	// Name is the header name of the field.
	Name string
	// Value is the value of the header or the entire field if the field is not a header.
	Value string
	// IsHeader is true if the field is a header, otherwise it is false.
	IsHeader bool
}

// ChangecaseAction changes the case of the field. If the field is a header, ChangecaseAction will change
// the case of the name and value components.
type ChangecaseAction struct {
	// Case can be one of the following:
	//   - "upper": changes the field to upper case
	//   - "lower": changes the field to lower case
	Case string
	// Next is the next action in the action tree.
	Next Action
}

// NewChangecaseAction returns a new ChangecaseAction with case c and next action n.
// If c is not "upper" or "lower", NewChangecaseAction returns an error.
func NewChangecaseAction(c string, next Action) (*ChangecaseAction, error) {
	if c != "upper" && c != "lower" {
		return nil, fmt.Errorf("invalid case: %s", c)
	}

	return &ChangecaseAction{
		Case: c,
		Next: terminateIfNil(next),
	}, nil
}

// String returns a string representation of the change case action.
func (a *ChangecaseAction) String() string {
	return fmt.Sprintf("changecase{%s}%s", a.Case, nextToString(a.Next))
}

// Apply changes the case of the field according to the case specified in the action. Apply calls
// the next action in the action tree.
func (a *ChangecaseAction) Apply(field Field) []Field {
	switch a.Case {
	case "upper":
		field.Name = strings.ToUpper(field.Name)
		field.Value = strings.ToUpper(field.Value)
	case "lower":
		field.Name = strings.ToLower(field.Name)
		field.Value = strings.ToLower(field.Value)
	}

	return a.Next.Apply(field)
}

// InsertAction inserts Value at Location in the Component of the field Num times.
type InsertAction struct {
	// Value is the value to insert into the field.
	Value string
	value string
	// Location can be one of the following:
	//	  - "start": inserts the value at the start of the field
	//   - "end": inserts the value at the end of the field
	//   - "mid": inserts the value at len(field)/2
	//   - "random": inserts the value at a random location, 0 < r < len(field), in the field.
	Location string
	// Component only applies if the field is a header, otherwise it is ignored and InsertAction is
	// applied to the entire field. Component can be one of the following:
	//   - "name": inserts the value in the name component of the header
	//	  - "value": inserts the value in the value component of the header
	Component string
	// Num is the number of times the value is inserted into the field. If Num is <= 0, Num is set to 1.
	Num int
	// Next is the next action in the action tree.
	Next Action
}

// NewInsertAction returns a new InsertAction with value v, location l, component c, number of
// copies of the value n, and next action. NewInsertAction returns an error if c is not "name" or "value"
// or if l is not "start", "end", "mid", or "random". If n is <= 0, n is set to 1.
func NewInsertAction(v, l, c string, n int, next Action) (*InsertAction, error) {
	if l != "start" && l != "end" && l != "mid" && l != "random" {
		return nil, fmt.Errorf("invalid location: %s", l)
	}

	if c != "name" && c != "value" {
		return nil, fmt.Errorf("invalid component: %s", c)
	}

	if n <= 0 {
		n = 1
	}

	nv := strings.Repeat(v, n)
	return &InsertAction{
		Value:     v,
		value:     nv,
		Location:  l,
		Component: c,
		Num:       n,
		Next:      terminateIfNil(next),
	}, nil
}

// String returns a string representation of the insert action.
func (a *InsertAction) String() string {
	return fmt.Sprintf("insert{%s:%s:%s:%d}%s", a.Value, a.Location, a.Component, a.Num, nextToString(a.Next))
}

// Apply inserts Value at Location in the Component of the field Num times. If the field is a header,
// Component is used to determine which component of the header to apply the action to. Apply calls
// the next action in the action tree.
func (a *InsertAction) Apply(field Field) []Field {
	field = modifyFieldComponent(field, a.Component, a.insert)
	return a.Next.Apply(field)
}

func (i *InsertAction) insert(str string) string {
	switch i.Location {
	case "start":
		return i.value + str
	case "end":
		return str + i.value
	case "mid":
		return str[:len(str)/2] + i.value + str[len(str)/2:]
	case "random":
		if len(str) <= 1 {
			return str
		}

		// get a random number between 1 and len(str)-1 to avoid inserting at the start or end of the string
		n := rand.Intn(len(str)-1) + 1
		return str[:n] + i.value + str[n:]
	default:
		return str
	}
}

// ReplaceAction replaces the field with Value in the Component of the field with Num copies of Value.
type ReplaceAction struct {
	// Value is the value to replace the field with.
	// Delete can be simulated by setting Value to an empty string.
	Value string
	value string
	// Component only applies if the field is a header, otherwise it is ignored and ReplaceAction is
	// applied to the entire field. Component can be one of the following:
	//   - "name": replaces the name component of the header with the value
	//   - "value": replaces the value component of the header with the value
	Component string
	// Num is the number of copies of Value to replace the field with. If Num is <= 0, Num is set to 1.
	Num int
	// Next is the next action in the action tree.
	Next Action
}

// NewReplaceAction returns a new ReplaceAction with value v, component c, number of copies of the value n,
// and next action. NewReplaceAction returns an error if c is not "name" or "value".
func NewReplaceAction(v, c string, n int, next Action) (*ReplaceAction, error) {
	if c != "name" && c != "value" {
		return nil, fmt.Errorf("invalid component: %s", c)
	}

	if n <= 0 {
		n = 1
	}

	nv := strings.Repeat(v, n)
	return &ReplaceAction{
		Value:     v,
		value:     nv,
		Component: c,
		Num:       n,
		Next:      terminateIfNil(next),
	}, nil
}

// String returns a string representation of the replace action.
func (a *ReplaceAction) String() string {
	return fmt.Sprintf("replace{%s:%s:%d}%s", a.Value, a.Component, a.Num, nextToString(a.Next))
}

// Apply replaces the field with Value in the Component of the field with Num copies of Value. Apply
// calls the next action in the action tree.
func (a *ReplaceAction) Apply(field Field) []Field {
	field = modifyFieldComponent(field, a.Component, func(s string) string {
		return a.value
	})

	return a.Next.Apply(field)
}

func modifyFieldComponent(field Field, component string, fn func(string) string) Field {
	if component == "name" && field.IsHeader {
		field.Name = fn(field.Name)
	} else {
		field.Value = fn(field.Value)
	}

	return field
}

// DuplicateAction duplicates the field and applies LeftAction to the original field and
// RightAction to the duplicate. The result of LeftAction and RightAction are concatenated and returned.
type DuplicateAction struct {
	// LeftAction is applied to the original field. If LeftAction is nil, the original field is unmodified.
	LeftAction Action
	// RightAction is applied to the duplicate field. If RightAction is nil, the duplicate field is unmodified.
	RightAction Action
}

// NewDuplicateAction returns a new DuplicateAction with left action l and right action r.
func NewDuplicateAction(l, r Action) *DuplicateAction {
	return &DuplicateAction{
		LeftAction:  terminateIfNil(l),
		RightAction: terminateIfNil(r),
	}
}

// String returns a string representation of the duplicate action.
func (a *DuplicateAction) String() string {
	return fmt.Sprintf("duplicate(%s, %s)", a.LeftAction, a.RightAction)
}

// Apply duplicates the field and applies LeftAction to the original field and RightAction to the duplicate.
func (a *DuplicateAction) Apply(field Field) []Field {
	f0 := a.LeftAction.Apply(field)
	f1 := a.RightAction.Apply(field)

	return append(f0, f1...)
}

// TerminateAction does not apply any modifications to the field or call another action.
// It is used to terminate the action chain.
type TerminateAction struct{}

// String returns a string representation of the return action. Which is an empty string.
func (a *TerminateAction) String() string {
	return ""
}

// Apply returns field.Name and field.Value concatenated together separated by ":" if field is a header.
// Otherwise, Apply returns field.Value. Apply does not call another action.
func (a *TerminateAction) Apply(field Field) []Field {
	return []Field{field}
}

// nextToString returns a string representation of the next action wrapped in parentheses following
// Geneva syntax.
func nextToString(next Action) string {
	if _, ok := next.(*TerminateAction); ok {
		return ""
	}

	return "(" + next.String() + ",)"
}

func terminateIfNil(a Action) Action {
	if a == nil {
		return &TerminateAction{}
	}

	return a
}
