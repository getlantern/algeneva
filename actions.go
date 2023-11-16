package algeneva

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
)

// action is an interface that all actions must implement.
type action interface {
	// string returns a string representation of the action in Geneva syntax as follows:
	//		<action>{<arg1>:<arg2>:...:<argn>}(<leftAction>,<rightAction>)
	//
	// The argument list maybe omitted if the action does not require any arguments. The left and right actions
	// are present only if there is another action in the action tree. If another action is present, it must be
	// formatted as (<leftAction>,<rightAction>), regardless of whether the left or right action is nil. In
	// which case, the action is formatted as (<leftAction>,) or (,<rightAction>).
	string() string
	// apply applies the action to the field and returns the result of the action.
	apply(fld field) []field
}

// newAction parses an action string in Geneva syntax and returns a ChangecaseAction, InsertAction, ReplaceAction,
// or DuplicateAction as an Action with the subsequent left and right action branches configured. If left or right
// is nil, the corresponding action is automatically set to TerminateAction. For ChangecaseAction, InsertAction,
// and ReplaceAction, left is configured as the next action. newAction returns an error if action is not a valid
// action or is formatted incorrectly.
func newAction(actionstr string, left, right action) (action, error) {
	br := strings.Index(actionstr, "{")
	var args []string
	if br != -1 {
		if actionstr[len(actionstr)-1] != '}' {
			return nil, errors.New("closing brace must end action string if args are given")
		}

		args = strings.Split(actionstr[br+1:len(actionstr)-1], ":")
		actionstr = actionstr[:br]
	}

	// only duplicate action supports a right branch action so return an error if the action is not duplicate and
	// the right action is not nil.
	if actionstr != "duplicate" && right != nil {
		return nil, fmt.Errorf("%s action does not support a right branch action", actionstr)
	}

	switch actionstr {
	case "changecase":
		if len(args) != 1 {
			return nil, errors.New("changecase requires 1 argument")
		}

		return newChangecaseAction(args[0], left)
	case "insert":
		if len(args) != 4 {
			return nil, errors.New("insert requires 4 arguments")
		}

		n, err := strconv.Atoi(args[3])
		if err != nil {
			return nil, fmt.Errorf("insert number of copies (%q) must be an int", args[3])
		}

		return newInsertAction(args[0], args[1], args[2], n, left)
	case "replace":
		if len(args) != 3 {
			return nil, errors.New("replace requires 3 arguments")
		}

		n, err := strconv.Atoi(args[2])
		if err != nil {
			return nil, fmt.Errorf("replace number of copies (%q) must be an int", args[2])
		}

		return newReplaceAction(args[0], args[1], n, left)
	case "duplicate":
		// duplicate action does not support arguments so return an error if the argument list is not empty
		if len(args) != 0 {
			return nil, errors.New("duplicate does not support arguments")
		}

		return newDuplicateAction(left, right), nil
	default:
		return nil, fmt.Errorf("unknown action: %s", actionstr)
	}
}

// field is the target field to apply an action to.
type field struct {
	// name is the header name of the field.
	name string
	// value is the value of the header or the entire field if the field is not a header.
	value string
	// isHeader is true if the field is a header, otherwise it is false.
	isHeader bool
}

// changecaseAction changes the case of the field. If the field is a header, changecaseAction will change
// the case of the name and value components.
type changecaseAction struct {
	// Case can be one of the following:
	//   - "upper": changes the field to upper case
	//   - "lower": changes the field to lower case
	Case string
	// next is the next action in the action tree.
	next action
}

// newChangecaseAction returns a new ChangecaseAction with case c and next action n. If next is nil, it is
// automatically set to TerminateAction. If c is not "upper" or "lower", newChangecaseAction returns an error.
func newChangecaseAction(c string, next action) (*changecaseAction, error) {
	if c != "upper" && c != "lower" {
		return nil, fmt.Errorf("invalid case: %s", c)
	}

	return &changecaseAction{
		Case: c,
		next: terminateIfNil(next),
	}, nil
}

// string returns a string representation of the change case action.
func (a *changecaseAction) string() string {
	return fmt.Sprintf("changecase{%s}%s", a.Case, nextToString(a.next))
}

// apply changes the case of the field according to the case specified in the action. apply calls
// the next action in the action tree.
func (a *changecaseAction) apply(fld field) []field {
	switch a.Case {
	case "upper":
		fld.name = strings.ToUpper(fld.name)
		fld.value = strings.ToUpper(fld.value)
	case "lower":
		fld.name = strings.ToLower(fld.name)
		fld.value = strings.ToLower(fld.value)
	}

	return a.next.apply(fld)
}

// insertAction inserts Value at Location in the Component of the field Num times.
type insertAction struct {
	// Value is the value to insert into the field. It is URL encoded with space encoded as %20 instead of "+".
	Value string
	value string
	// location can be one of the following:
	//   - "start": inserts the value at the start of the field
	//   - "end": inserts the value at the end of the field
	//   - "mid": inserts the value at len(field)/2
	//   - "random": inserts the value at a random location, 0 < r < len(field), in the field.
	location string
	// component only applies if the field is a header, otherwise it is ignored and InsertAction is
	// applied to the entire field. component can be one of the following:
	//   - "name": inserts the value in the name component of the header
	//   - "value": inserts the value in the value component of the header
	component string
	// num is the number of times the value is inserted into the field. If num is <= 0, num is set to 1.
	num int
	// next is the next action in the action tree.
	next action
}

// newInsertAction returns a new InsertAction with value v, location l, component c, number of copies of the value n,
// and next action. If next is nil, it is automatically set to TerminateAction. newInsertAction returns an error if c
// is not "name" or "value" or if l is not "start", "end", "mid", or "random". If n is <= 0, n is set to 1.
func newInsertAction(v, l, c string, n int, next action) (*insertAction, error) {
	if l != "start" && l != "end" && l != "mid" && l != "random" {
		return nil, fmt.Errorf("invalid location: %s", l)
	}

	if c != "name" && c != "value" {
		return nil, fmt.Errorf("invalid component: %s", c)
	}

	if n <= 0 {
		n = 1
	}

	// geneva uses URL encoding for the value but with %20 as space instead of +, so we need to unescape it
	nv, err := url.PathUnescape(v)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s, %w", v, err)
	}

	nv = strings.Repeat(nv, n)
	return &insertAction{
		Value:     v,
		value:     nv,
		location:  l,
		component: c,
		num:       n,
		next:      terminateIfNil(next),
	}, nil
}

// string returns a string representation of the insert action.
func (a *insertAction) string() string {
	return fmt.Sprintf("insert{%s:%s:%s:%d}%s", a.Value, a.location, a.component, a.num, nextToString(a.next))
}

// apply inserts Value at Location in the Component of the field Num times. If the field is a header,
// Component is used to determine which component of the header to apply the action to. apply calls
// the next action in the action tree.
func (a *insertAction) apply(fld field) []field {
	fld = modifyFieldComponent(fld, a.component, a.insert)
	return a.next.apply(fld)
}

func (i *insertAction) insert(str string) string {
	switch i.location {
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

// replaceAction replaces the field with Value in the Component of the field with Num copies of Value.
type replaceAction struct {
	// Value is the value to replace the field with. It is URL encoded with space encoded as %20 instead of "+".
	// Delete can be simulated by setting Value to an empty string.
	Value string
	value string
	// component only applies if the field is a header, otherwise it is ignored and ReplaceAction is
	// applied to the entire field. component can be one of the following:
	//   - "name": replaces the name component of the header with the value
	//   - "value": replaces the value component of the header with the value
	component string
	// num is the number of copies of Value to replace the field with. If num is <= 0, num is set to 1.
	num int
	// next is the next action in the action tree.
	next action
}

// newReplaceAction returns a new ReplaceAction with value v, component c, number of copies of the value n, and next
// action. If next is nil, it is automatically set to TerminateAction. newReplaceAction returns an error if c is not
// "name" or "value".
func newReplaceAction(v, c string, n int, next action) (*replaceAction, error) {
	if c != "name" && c != "value" {
		return nil, fmt.Errorf("invalid component: %s", c)
	}

	if n <= 0 {
		n = 1
	}

	// geneva uses URL encoding for the value but with %20 as space instead of +, so we need to unescape it
	nv, err := url.PathUnescape(v)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s, %w", v, err)
	}

	nv = strings.Repeat(nv, n)
	return &replaceAction{
		Value:     v,
		value:     nv,
		component: c,
		num:       n,
		next:      terminateIfNil(next),
	}, nil
}

// string returns a string representation of the replace action.
func (a *replaceAction) string() string {
	return fmt.Sprintf("replace{%s:%s:%d}%s", a.Value, a.component, a.num, nextToString(a.next))
}

// apply replaces the field with Value in the Component of the field with Num copies of Value. apply
// calls the next action in the action tree.
func (a *replaceAction) apply(fld field) []field {
	fld = modifyFieldComponent(fld, a.component, func(s string) string {
		return a.value
	})

	return a.next.apply(fld)
}

func modifyFieldComponent(fld field, component string, fn func(string) string) field {
	if component == "name" && fld.isHeader {
		fld.name = fn(fld.name)
	} else {
		fld.value = fn(fld.value)
	}

	return fld
}

// duplicateAction duplicates the field and applies LeftAction to the original field and
// RightAction to the duplicate. The result of LeftAction and RightAction are concatenated and returned.
type duplicateAction struct {
	// leftAction is applied to the original field.
	leftAction action
	// rightAction is applied to the duplicate field.
	rightAction action
}

// newDuplicateAction returns a new DuplicateAction with left action l and right action r.
// If l or r is nil, newDuplicateAction automatically sets the action to TerminateAction.
func newDuplicateAction(l, r action) *duplicateAction {
	return &duplicateAction{
		leftAction:  terminateIfNil(l),
		rightAction: terminateIfNil(r),
	}
}

// string returns a string representation of the duplicate action.
func (a *duplicateAction) string() string {
	return fmt.Sprintf("duplicate(%s, %s)", a.leftAction.string(), a.rightAction.string())
}

// apply duplicates the field and applies LeftAction to the original field and RightAction to the duplicate.
func (a *duplicateAction) apply(fld field) []field {
	f0 := a.leftAction.apply(fld)
	f1 := a.rightAction.apply(fld)

	return append(f0, f1...)
}

// terminateAction does not apply any modifications to the field or call another action.
// It is used to terminate the action chain.
type terminateAction struct{}

// string returns a string representation of the return action. Which is an empty string.
func (a *terminateAction) string() string {
	return ""
}

// apply returns field.Name and field.Value concatenated together separated by ":" if field is a header.
// Otherwise, apply returns field.Value. apply does not call another action.
func (a *terminateAction) apply(fld field) []field {
	return []field{fld}
}

// nextToString returns a string representation of the next action wrapped in parentheses following
// Geneva syntax.
func nextToString(next action) string {
	if _, ok := next.(*terminateAction); ok {
		return ""
	}

	return "(" + next.string() + ",)"
}

func terminateIfNil(a action) action {
	if a == nil {
		return &terminateAction{}
	}

	return a
}
