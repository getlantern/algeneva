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
	// The argument list may be omitted if the action does not require any arguments. The left and
	// right actions are present only if there is another action in the action tree. If another
	// action is present, it must be formatted as (<leftAction>,<rightAction>), regardless of
	// whether the left or right action is nil. In which case, the action is formatted as
	// (<leftAction>,) or (,<rightAction>).
	string() string
	// apply applies the action to fld and returns the result of the action.
	apply(fld field) []field
}

// newAction parses an action string in Geneva syntax and returns the corresponding action. If left
// or right is nil, they are automatically set to terminateAction. Duplicate is the only action
// that supports a right action. All other actions will use left as the next action in the action
// chain. newAction returns an error if action is not a valid action or is formatted incorrectly.
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

	// only duplicate action supports a right branch action so return an error if the action is not
	// duplicate and the right action is not nil or terminate.
	if actionstr != "duplicate" && right != nil {
		if _, ok := right.(*terminateAction); !ok {
			return nil, fmt.Errorf(
				"%s action does not support a right branch action (%s)",
				actionstr,
				right.string(),
			)
		}
	}

	switch actionstr {
	case "changecase":
		if len(args) != 1 {
			return nil, errors.New("changecase requires 1 argument")
		}

		return newChangecaseAction(args[0], left)
	case "insert":
		var n int
		switch len(args) {
		case 3: // default to 1
			n = 1
		case 4: // number of copies is present
			if args[3] != "" {
				var err error
				if n, err = strconv.Atoi(args[3]); err != nil {
					return nil, fmt.Errorf("insert number of copies (%q) must be an int", args[3])
				}
			}
		default:
			return nil, errors.New("insert requires 3 or 4 arguments. 'num' is optional and defaults to 1")
		}

		return newInsertAction(args[0], args[1], args[2], n, left)
	case "replace":
		var n int
		switch len(args) {
		case 2: // default to 1
			n = 1
		case 3: // number of copies is present
			if args[2] != "" {
				var err error
				if n, err = strconv.Atoi(args[2]); err != nil {
					return nil, fmt.Errorf("replace number of copies (%q) must be an int", args[2])
				}
			}
		default:
			return nil, errors.New("replace requires 2 or 3 arguments. 'num' is optional and defaults to 1")
		}

		return newReplaceAction(args[0], args[1], n, left)
	case "duplicate":
		// duplicate action does not support arguments so return an error if the argument list is
		// not empty
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
	// name is the header name if the field is a header.
	name string
	// value is the value of the header or the entire field if the field is not a header.
	value    string
	isHeader bool
}

// changecaseAction changes the case of the field. If the field is a header, changecaseAction will
// change the case of both the name and value components.
type changecaseAction struct {
	// toCase can be one of the following:
	//   - "upper": changes the field to upper case
	//   - "lower": changes the field to lower case
	toCase string
	// next is the next action in the action tree.
	next action
}

// newChangecaseAction returns a new changecaseAction with toCase and next action. If next is nil,
// it is automatically set to terminateAction. newChangecaseAction returns an error if toCase is
// invalid.
func newChangecaseAction(toCase string, next action) (*changecaseAction, error) {
	if toCase != "upper" && toCase != "lower" {
		return nil, fmt.Errorf("invalid case: %s", toCase)
	}

	return &changecaseAction{
		toCase: toCase,
		next:   terminateIfNil(next),
	}, nil
}

func (a *changecaseAction) string() string {
	return fmt.Sprintf("changecase{%s}%s", a.toCase, nextToString(a.next))
}

// apply applies the changecase action to fld and calls the next action in the action tree.
func (a *changecaseAction) apply(fld field) []field {
	switch a.toCase {
	case "upper":
		fld.name = strings.ToUpper(fld.name)
		fld.value = strings.ToUpper(fld.value)
	case "lower":
		fld.name = strings.ToLower(fld.name)
		fld.value = strings.ToLower(fld.value)
	}

	return a.next.apply(fld)
}

// insertAction inserts value at location in the field component num times.
type insertAction struct {
	// value is the value to insert into the field. It is URL percent encoded.
	value    string
	newValue string
	// location can be one of the following:
	//   - "start": inserts the value at the start of the field
	//   - "end": inserts the value at the end of the field
	//   - "middle": inserts the value at len(field)/2
	//   - "random": inserts the value at a random location, 0 < r < len(field), in the field.
	location string
	// component only applies if the field is a header, otherwise it is ignored and insertAction is
	// applied to the entire field. component can be one of the following:
	//   - "name": inserts the value in the name component of the header
	//   - "value": inserts the value in the value component of the header
	component string
	// num is the number of times the value is inserted into the field.
	num int
	// next is the next action in the action tree.
	next action
}

// newInsertAction returns a new insertAction with value, location, component, number of
// copies of the value, and next action. If next is nil, it is automatically set to
// terminateAction. If num <= 0, num is set to 1. newInsertAction returns an error if component or
// location are invalid.
func newInsertAction(value, location, component string, num int, next action) (*insertAction, error) {
	if location != "start" && location != "end" && location != "middle" && location != "random" {
		return nil, fmt.Errorf("invalid location: %s", location)
	}

	if component != "name" && component != "value" {
		return nil, fmt.Errorf("invalid component: %s", component)
	}

	if num <= 0 {
		num = 1
	}

	// geneva uses URL percent encoding for the value, so we need to unescape it
	nv, err := url.PathUnescape(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s, %w", value, err)
	}

	nv = strings.Repeat(nv, num)
	return &insertAction{
		value:     value,
		newValue:  nv,
		location:  location,
		component: component,
		num:       num,
		next:      terminateIfNil(next),
	}, nil
}

func (a *insertAction) string() string {
	return fmt.Sprintf("insert{%s:%s:%s:%d}%s", a.value, a.location, a.component, a.num, nextToString(a.next))
}

// apply applies the insert action to fld and calls the next action in the action tree.
func (a *insertAction) apply(fld field) []field {
	fld = modifyFieldComponent(fld, a.component, a.insert)
	return a.next.apply(fld)
}

func (i *insertAction) insert(str string) string {
	switch i.location {
	case "start":
		return i.newValue + str
	case "end":
		return str + i.newValue
	case "middle":
		return str[:len(str)/2] + i.newValue + str[len(str)/2:]
	case "random":
		if len(str) <= 1 {
			return str
		}

		// get a random number between 1 and len(str)-1, inclusive. We don't want to insert at the
		// start or end of the field, otherwise, we would have used the start or end as location.
		// This contraint is enforced by the geneva specification.
		n := rand.Intn(len(str)-1) + 1
		return str[:n] + i.newValue + str[n:]
	default:
		return str
	}
}

// replaceAction replaces the field component with num copies of value.
type replaceAction struct {
	// value is the value to replace the field with. It is URL percent encoded. Delete can be
	// simulated by setting value to an empty string.
	value string
	// newValue is the unescaped newValue of value repeated num times. This is used to avoid
	// recomputing it for each application.
	newValue string
	// component only applies if the field is a header, otherwise it is ignored and replaceAction is
	// applied to the entire field. component can be one of the following:
	//   - "name": replaces the name component of the header with the newValue
	//   - "value": replaces the value component of the header with the newValue
	component string
	// num is the number of copies of value to replace the field with.
	num int
	// next is the next action in the action tree.
	next action
}

// newReplaceAction returns a new replaceAction with value, component, number of copies of the
// value, and next action. If next is nil, it is automatically set to terminateAction. If num <= 0,
// num is set to 1. newReplaceAction returns an error if component is invalid.
func newReplaceAction(value, component string, num int, next action) (*replaceAction, error) {
	if component != "name" && component != "value" {
		return nil, fmt.Errorf("invalid component: %s", component)
	}

	if num <= 0 {
		num = 1
	}

	// geneva uses URL percent encoding for the value so we need to unescape it
	newValue, err := url.PathUnescape(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s, %w", value, err)
	}

	newValue = strings.Repeat(newValue, num)
	return &replaceAction{
		value:     value,
		newValue:  newValue,
		component: component,
		num:       num,
		next:      terminateIfNil(next),
	}, nil
}

func (a *replaceAction) string() string {
	return fmt.Sprintf("replace{%s:%s:%d}%s", a.value, a.component, a.num, nextToString(a.next))
}

// apply applies the replace action to fld and calls the next action in the action tree.
func (a *replaceAction) apply(fld field) []field {
	fld = modifyFieldComponent(fld, a.component, func(s string) string {
		return a.newValue
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

// duplicateAction duplicates the field and applies leftAction to the original field and
// rightAction to the duplicate.
type duplicateAction struct {
	leftAction  action
	rightAction action
}

// newDuplicateAction returns a new duplicateAction with left and right actions. newDuplicateAction
// automatically sets left and right to terminateAction if they are nil.
func newDuplicateAction(left, right action) *duplicateAction {
	return &duplicateAction{
		leftAction:  terminateIfNil(left),
		rightAction: terminateIfNil(right),
	}
}

func (a *duplicateAction) string() string {
	return fmt.Sprintf("duplicate(%s, %s)", a.leftAction.string(), a.rightAction.string())
}

// apply duplicates fld, applies leftAction and rightAction, and returns the concatenated results.
func (a *duplicateAction) apply(fld field) []field {
	f0 := a.leftAction.apply(fld)
	f1 := a.rightAction.apply(fld)

	return append(f0, f1...)
}

// terminateAction does not apply any modifications to the field or call another action.
// It is used to terminate the action chain.
type terminateAction struct{}

func (a *terminateAction) string() string { return "" }

// apply returns the field without any modifications as a []field.
func (a *terminateAction) apply(fld field) []field { return []field{fld} }

// nextToString returns a string representation of next following Geneva syntax.
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
