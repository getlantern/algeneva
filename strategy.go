package geneva

import (
	"bytes"
	"fmt"
	"strings"
)

var ErrInvalidRule = fmt.Errorf("invalid rule")

// NewStrategy constructs a Strategy from strategy. strategy consists of a series of rules separated by '|'. Each rule
// is formatted as '<trigger>-<action>-|'. An error is returned if strategy is not a valid strategy or is formatted
// incorrectly.
func NewStrategy(strategy string) (Strategy, error) {
	return parseStrategy(strategy)
}

// parseStrategy parses a string, strategy, and returns a Strategy. It returns an error if strategy is not a valid
// strategy or is formatted incorrectly. A valid strategy is formatted as '<trigger>-<action>-|'.
func parseStrategy(strategy string) (Strategy, error) {
	var rules []Rule

	// Split the string into rules, which are separated by '|', and parse each rule.
	for _, rule := range strings.Split(strategy, "|") {
		r, err := parseRule(rule)
		if err != nil {
			return Strategy{}, err
		}

		rules = append(rules, r)
	}

	return Strategy{
		Rules: rules,
	}, nil
}

// parseRule parses a string, rule, and returns a Rule. It returns an error if rule is not a valid rule or is
// formatted incorrectly.
func parseRule(s string) (Rule, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return Rule{}, fmt.Errorf("%w: %s, should be formatted as '<trigger>-<actions>-|'", ErrInvalidRule, s)
	}

	trigger, err := parseTrigger(parts[0])
	if err != nil {
		return Rule{}, err
	}

	tree, err := parseAction(parts[1])
	if err != nil {
		return Rule{}, err
	}

	return Rule{
		Trigger: trigger,
		Tree:    tree,
	}, nil
}

// parseTrigger parses a string, trigger, and returns a Trigger. It returns an error if trigger is not a valid trigger
// or is formatted incorrectly. A valid trigger is formatted as '[<proto>:<field>:<matchstr>]', where proto is the
// protocol, field is the target field to apply actions, and matchstr is the string to match against.
// Currently only HTTP is supported as a protocol.
func parseTrigger(trigger string) (Trigger, error) {
	parts := strings.Split(trigger, ":")

	// Check if the trigger is formatted correctly and not empty.
	if trigger == "" ||
		trigger[0] != '[' ||
		trigger[len(trigger)-1] != ']' ||
		len(parts) != 3 {
		return Trigger{},
			fmt.Errorf("invalid trigger: %s, should be formatted as '[<proto>:<field>:<matchstr>]'", trigger)
	}

	proto := strings.ToUpper(parts[0][1:])
	switch proto {
	case "HTTP":
	case "DNS", "DNSQR":
		return Trigger{}, fmt.Errorf("DNS and DNSQR are not supported yet")
	default:
		return Trigger{}, fmt.Errorf("unsupported protocol: %s", proto)
	}

	field := strings.ToLower(parts[1])
	matchstr := strings.ToLower(parts[2])

	return Trigger{
		Proto:       proto,
		TargetField: field,
		MatchStr:    matchstr,
	}, nil
}

// parseAction parses a string, action, and returns an Action. It returns an error if action is not a valid action or
// is formatted incorrectly. A valid action is formatted as '<action>[(<left>,<right>)]', where left and right are
// subsequences of actions. '(<left>,<right>)' is only required if there is a subsequent action.
func parseAction(action string) (Action, error) {
	if action == "" {
		return &TerminateAction{}, nil
	}

	// check is there is a next action by finding the first and last parentheses.
	fp := strings.Index(action, "(")
	lp := strings.LastIndex(action, ")")
	if lp < fp {
		return nil, ErrInvalidRule
	}

	// if we didn't find any parentheses, then there is no next action, so just construct the action.
	if fp == -1 {
		return NewAction(action, nil)
	}

	// there is a next action, so we need to split what's inside the parentheses into the left and right subactions.
	l, r, err := splitLeftRight(action[fp : lp+1])
	if err != nil {
		return nil, err
	}

	// parse the left action.
	left, err := parseAction(l)
	if err != nil {
		return nil, err
	}

	// construct the action with left as the next action since only duplicate can have a right action.
	a, err := NewAction(action[:fp], left)
	if err != nil {
		return nil, err
	}

	// if the action is a duplicate, then parse the right action.
	dp, ok := a.(*DuplicateAction)
	if ok {
		if dp.RightAction, err = parseAction(r); err != nil {
			return nil, err
		}

		return dp, nil
	}

	// if the action is not a duplicate, then there should not be a right action.
	if r != "" {
		return nil,
			fmt.Errorf("%w: %s, only duplicate action support second subsequent action", ErrInvalidRule, action)
	}

	return a, nil
}

// splitLeftRight splits action into the left and right subactions. action is the next action in the action tree, and
// is formatted as '([<leftaction>],[<rightaction>])' where leftaction and rightaction can be subsequences of actions.
func splitLeftRight(action string) (string, string, error) {
	// since the comma must be present, there is at least one for each '('. so we just need to count the ',' and '('
	// as we iterate over the string until the counts are equal. everything before the ',' must be the left action,
	// and everything after must be the right action.
	var np, nc int
	for i, c := range action {
		switch c {
		case '(':
			np++
		case ',':
			nc++
		}

		if np == nc {
			return action[1:i], action[i+1 : len(action)-1], nil
		}
	}

	// if we exit the loop, then the action is not formatted correctly.
	return "", "", ErrInvalidRule
}

// Strategy is a series of Geneva rules to be applied to a request.
type Strategy struct {
	Rules []Rule
}

// String returns a string representation of the Strategy.
func (s *Strategy) String() string {
	var rules []string
	for _, rule := range s.Rules {
		rules = append(rules, rule.String())
	}
	return strings.Join(rules, "")
}

// Apply applies the strategy to the request.
func (s *Strategy) Apply(req []byte) ([]byte, error) {
	r, err := newRequest(req)
	if err != nil {
		return req, err
	}

	// iterate over each rule and if the trigger matches, apply the action tree to the target field.
	for _, rule := range s.Rules {
		if field, match := rule.Trigger.Match(r); match {
			// apply the action tree to the target field.
			// since the duplicate action can cause the tree to branch, the modifications are returned as a slice of
			// Fields which need to be applied to the request.
			mods := rule.Apply(field)
			// apply the modifications to the request.
			r.applyModifications(field, mods)
		}
	}

	return r.bytes(), nil
}

// Rule is a single trigger and action tree to be applied to the target field if the trigger is met.
type Rule struct {
	// Trigger is the condition that must be met for the rule to be applied.
	Trigger Trigger
	// Tree is the action tree to be applied to the target field if the trigger is met.
	Tree Action
}

// String returns a string representation of the Rule.
func (r *Rule) String() string {
	return fmt.Sprintf("%s-%s-|", r.Trigger, r.Tree)
}

// Apply applies the Tree to the field.
func (r *Rule) Apply(field Field) []Field {
	return r.Tree.Apply(field)
}

// Trigger is a condition that must be met for the rule to be applied.
type Trigger struct {
	// Proto is the protocol of the request.
	Proto string
	// TargetField is the field to apply actions.
	TargetField string
	// MatchStr is the value Field needs to be to match. If MatchStr is '*', then the trigger will always match.
	MatchStr string
}

// String returns a string representation of the Trigger.
func (t *Trigger) String() string {
	return fmt.Sprintf("[%s:%s:%s]", strings.ToUpper(t.Proto), t.TargetField, t.MatchStr)
}

// Match returns whether the value of TargetField of req matches MatchStr. If true, the target field is returned
// as a Field.
// Since DNS and DNSQR are not supported yet, Proto is ignored, except if it is empty, in which case it will fail.
func (t *Trigger) Match(req *request) (Field, bool) {
	if t.Proto == "" {
		return Field{}, false
	}

	field := Field{}
	switch t.TargetField {
	case "method":
		field.Value = req.method
	case "path":
		field.Value = req.path
	case "version":
		field.Value = req.version
	default:
		// the target field is a header. find it and parse it into a Field.
		hidx := strings.Index(req.headers, t.TargetField)
		if hidx == -1 {
			// the header does not exist.
			return Field{}, false
		}

		nlidx := strings.Index(req.headers[hidx:], "\n")
		if nlidx == -1 {
			nlidx = len(req.headers[hidx:])
		}

		header := req.headers[hidx : hidx+nlidx]
		parts := strings.Split(strings.TrimSpace(header), ":")
		field = Field{
			Name:     parts[0],
			Value:    parts[1],
			IsHeader: true,
		}
	}

	return field, matchValue(field.Value, t.MatchStr)
}

func matchValue(value, matchstr string) bool {
	return matchstr == "*" || value == matchstr
}

// request is an extremely simple HTTP request parser. It only parses the method, path, and version from the start
// line, and separates the headers and body. It does not parse the headers or body.
type request struct {
	method  string
	path    string
	version string
	headers string
	body    []byte
}

// newRequest parses a byte slice, req, into a request. It returns an error if req is not a valid HTTP request.
func newRequest(req []byte) (*request, error) {
	// Find the index of the end of the headers.
	idx := bytes.Index(req, []byte("\r\n\r\n"))
	if idx == -1 {
		return nil, fmt.Errorf("invalid request: %s", req)
	}

	// Split the request into the start line, headers, and body.
	startLine, headers, _ := bytes.Cut(req[:idx], []byte("\r\n"))
	// Split the start line into the method, path, and version.
	mpv := strings.Split(string(startLine), " ")
	if len(mpv) != 3 {
		return nil, fmt.Errorf("invalid request: %s", req)
	}

	return &request{
		method:  mpv[0],
		path:    mpv[1],
		version: mpv[2],
		headers: string(headers),
		body:    req[idx+4:],
	}, nil
}

// applyModifications applies the modifications, mods, to the field in the request. field is the original unmodified
// field.
func (r *request) applyModifications(field Field, mods []Field) {
	// iterate over mods and construct the new value.
	var newValue string
	for _, mod := range mods {
		if mod.IsHeader {
			newValue += fmt.Sprintf("%s: %s\r\n", mod.Name, mod.Value)
		} else {
			newValue += mod.Value
		}
	}

	switch field.Name {
	case "method":
		r.method = newValue
	case "path":
		r.path = newValue
	case "version":
		r.version = newValue
	default:
		h := field.Name + ": " + newValue
		r.headers = strings.Replace(r.headers, h, newValue, 1)
	}
}

// bytes merges the head and body of the request back into a []byte and returns it.
func (r *request) bytes() []byte {
	head := fmt.Sprintf("%s %s %s\r\n%s\r\n\r\n", r.method, r.path, r.version, r.headers)

	size := len(head) + len(r.body)
	buf := make([]byte, 0, size)

	copy(buf, head)
	copy(buf[len(head):], r.body)

	return buf
}
