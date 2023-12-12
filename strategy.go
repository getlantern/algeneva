package algeneva

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidRule is returned when a rule is not a valid rule or is formatted incorrectly.
	ErrInvalidRule = errors.New("invalid rule")
	// ErrInvalidAction is returned when an action is not a valid action or is formatted incorrectly.
	ErrInvalidAction = errors.New("invalid action")
)

// HTTPStrategy is a series of Geneva rules to be applied to a request.
type HTTPStrategy struct {
	rules []rule
}

// NewHTTPStrategy constructs a HTTP Strategy from strategystr. strategystr consists of a series of rules separated by
// '|'. Each rule is formatted as '<trigger>-<action>-|', rules must end with '-|'. An error is returned if
// strategystr is not a valid strategy or is formatted incorrectly.
func NewHTTPStrategy(strategystr string) (HTTPStrategy, error) {
	var rules []rule

	// Split the string into rules, which are separated by '|', and parse each rule.
	parts := strings.SplitAfter(strategystr, "|")
	switch {
	case parts[len(parts)-1] != "":
		return HTTPStrategy{}, fmt.Errorf("%w: %s, rules must end with '-|'", ErrInvalidRule, strategystr)
	case parts[0] == "":
		return HTTPStrategy{}, errors.New("no rules found")
	default:
	}

	// The last element will be empty since each rule always ends with '|', so we ignore it.
	for _, rule := range parts[:len(parts)-1] {
		r, err := parseRule(rule)
		if err != nil {
			return HTTPStrategy{}, err
		}

		rules = append(rules, r)
	}

	return HTTPStrategy{
		rules: rules,
	}, nil
}

// string returns a string representation of the Strategy.
func (s *HTTPStrategy) string() string {
	var rules []string
	for _, r := range s.rules {
		rules = append(rules, r.string())
	}

	return strings.Join(rules, "")
}

// Apply applies the strategy to the input HTTP request. An error is returned
// if the input does not represent an HTTP request. The input does not need to
// include the body, but must include the start-line and all header lines. The
// body may be included, in which case it will be included in the return value,
// unmodified.
func (s *HTTPStrategy) Apply(req []byte) ([]byte, error) {
	r, err := newRequest(req)
	if err != nil {
		return req, err
	}

	s.apply(r)
	return r.bytes(), nil
}

// apply applies the strategy to the request.
func (s *HTTPStrategy) apply(req *request) {
	// iterate over each rule and if the trigger matches, apply the action tree to the target field.
	for _, r := range s.rules {
		if fld, match := r.trigger.match(req); match {
			// apply the action tree to the target field.
			// since the duplicate action can cause the tree to branch, the modifications are returned as a slice of
			// Fields which need to be applied to the request.
			mods := r.apply(fld)
			// apply the modifications to the request.
			applyModifications(req, fld, mods)
		}
	}
}

// rule is a single trigger and action tree to be applied to the target field if the trigger is met.
type rule struct {
	// trigger is the condition that must be met for the rule to be applied.
	trigger trigger
	// tree is the action tree to be applied to the target field if the trigger is met.
	tree action
}

// string returns a string representation of the Rule.
func (r *rule) string() string {
	return fmt.Sprintf("%s-%s-|", r.trigger.string(), r.tree.string())
}

// apply applies the Tree to the field.
func (r *rule) apply(f field) []field {
	return r.tree.apply(f)
}

// trigger is a condition that must be met for the rule to be applied.
type trigger struct {
	// proto is the protocol of the request.
	proto string
	// targetField is the field to apply actions.
	targetField string
	// matchStr is the value Field needs to be to match. If matchStr is '*', then the trigger will always match.
	matchStr string
}

// string returns a string representation of the Trigger.
func (t *trigger) string() string {
	return fmt.Sprintf("[%s:%s:%s]", strings.ToUpper(t.proto), t.targetField, t.matchStr)
}

// match returns whether the value of TargetField of req matches MatchStr. If true, the target field is returned
// as a Field.
// Since DNS and DNSQR are not supported yet, Proto is ignored, except if it is empty, in which case it will fail.
func (t *trigger) match(req *request) (field, bool) {
	if t.proto == "" {
		return field{}, false
	}

	var fld field
	switch t.targetField {
	case "method":
		fld = field{
			name:  "method",
			value: req.method,
		}
	case "path":
		fld = field{
			name:  "path",
			value: req.path,
		}
	case "version":
		fld = field{
			name:  "version",
			value: req.version,
		}
	default:
		// the target field is a header. find it and parse it into a Field.
		header := req.getHeader(t.targetField)
		if header == "" {
			return field{}, false
		}

		parts := strings.Split(header, ":")
		fld = field{
			name:     parts[0],
			value:    parts[1],
			isHeader: true,
		}
	}

	return fld, matchValue(fld.value, t.matchStr)
}

func matchValue(value, matchstr string) bool {
	return matchstr == "*" || value == matchstr
}

// parseRule parses a string, rule, and returns a Rule. It returns an error if rule is not a valid rule or is
// formatted incorrectly.
func parseRule(r string) (rule, error) {
	parts := strings.Split(r, "-")

	if len(parts) != 3 && parts[len(parts)-1] != "|" {
		return rule{}, fmt.Errorf("%w: %s, should be formatted as '<trigger>-<actions>-|'", ErrInvalidRule, r)
	}

	trig, err := parseTrigger(parts[0])
	if err != nil {
		return rule{}, err
	}

	tree, err := parseAction(parts[1])
	if err != nil {
		return rule{}, err
	}

	return rule{
		trigger: trig,
		tree:    tree,
	}, nil
}

// parseTrigger parses a string, trigger, and returns a Trigger. It returns an error if trigger is not a valid trigger
// or is formatted incorrectly. A valid trigger is formatted as '[<proto>:<field>:<matchstr>]', where proto is the
// protocol, field is the target field to apply actions, and matchstr is the string to match against.
// Currently only HTTP is supported as a protocol.
func parseTrigger(str string) (trigger, error) {
	parts := strings.Split(str, ":")

	// Check if the trigger is formatted correctly and not empty.
	if str == "" ||
		str[0] != '[' ||
		str[len(str)-1] != ']' ||
		len(parts) != 3 {
		return trigger{},
			fmt.Errorf("%w: %s, trigger should be formatted as '[<proto>:<field>:<matchstr>]'", ErrInvalidRule, str)
	}

	proto := strings.ToUpper(parts[0][1:])
	switch proto {
	case "HTTP":
	case "DNS", "DNSQR":
		return trigger{}, fmt.Errorf("%w: trigger protocols DNS and DNSQR are not supported yet", ErrInvalidRule)
	default:
		return trigger{}, fmt.Errorf("%w: unsupported trigger protocol %q", ErrInvalidRule, proto)
	}

	fld := strings.ToLower(parts[1])
	matchstr := strings.ToLower(parts[2][:len(parts[2])-1])

	return trigger{
		proto:       proto,
		targetField: fld,
		matchStr:    matchstr,
	}, nil
}

// parseAction parses an action string in Geneva syntax and returns an Action. It returns an error if action is not a valid action or
// is formatted incorrectly. A valid action is formatted as '<action>[(<left>,<right>)]', where left and right are
// subsequences of actions. '(<left>,<right>)' is only required if there is a subsequent action.
func parseAction(actionstr string) (action, error) {
	if actionstr == "" {
		return &terminateAction{}, nil
	}

	// check is there is a next action by finding the first and last parentheses.
	fp := strings.Index(actionstr, "(")
	lp := strings.LastIndex(actionstr, ")")
	if lp < fp {
		return nil, fmt.Errorf("%w: %s, missing matching parentheses", ErrInvalidRule, actionstr)
	}

	// if we didn't find any parentheses, then there is no next action, so just construct the action.
	if fp == -1 {
		a, err := newAction(actionstr, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: %s, %s", ErrInvalidAction, actionstr, err)
		}

		return a, nil
	}

	// there is a next action, so we need to split what's inside the parentheses into the left and right actions.
	l, r, err := splitLeftRight(actionstr[fp : lp+1])
	if err != nil {
		return nil, err
	}

	// parse the left and right actions.
	var left, right, a action
	if left, err = parseAction(l); err != nil {
		return nil, err
	}

	if right, err = parseAction(r); err != nil {
		return nil, err
	}

	// construct the action
	if a, err = newAction(actionstr[:fp], left, right); err != nil {
		return nil, fmt.Errorf("%w: %s, %s", ErrInvalidAction, actionstr, err)
	}

	return a, nil
}

// splitLeftRight splits action into the left and right subactions. action is the next action in the action tree, and
// is formatted as '([<leftaction>],[<rightaction>])' where leftaction and rightaction can be subsequences of actions.
func splitLeftRight(actionstr string) (string, string, error) {
	// since the comma must be present, there is at least one for each '('. so we just need to count the ',' and '('
	// as we iterate over the string until the counts are equal. everything before the ',' must be the left action,
	// and everything after must be the right action.
	var np, nc int
	for i, c := range actionstr {
		switch c {
		case '(':
			np++
		case ',':
			nc++
		}

		if np == nc {
			return actionstr[1:i], actionstr[i+1 : len(actionstr)-1], nil
		}
	}

	// if we exit the loop, then the action is not formatted correctly.
	return "", "", fmt.Errorf("%w: invalid format for left and right actions from %s", ErrInvalidRule, actionstr)
}

// applyModifications applies the modifications, mods, to the field in the request. field is the original unmodified
// field.
func applyModifications(req *request, fld field, mods []field) {
	// iterate over mods and construct the new value.
	var newValue string
	if fld.isHeader {
		var vals []string
		for _, mod := range mods {
			vals = append(vals, mod.name+":"+mod.value)
		}

		newValue = strings.Join(vals, "\r\n")
	} else {
		for _, mod := range mods {
			newValue += mod.value
		}
	}

	switch fld.name {
	case "method":
		req.method = newValue
	case "path":
		req.path = newValue
	case "version":
		req.version = newValue
	default:
		h := fld.name + ":" + fld.value
		req.headers = strings.Replace(req.headers, h, newValue, 1)
	}
}
