package algeneva

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidRule is returned when a rule is not a valid rule or is formatted incorrectly.
	ErrInvalidRule = errors.New("invalid rule")
	// ErrInvalidRule is returned when a rule is not a valid rule or is formatted incorrectly.
	ErrInvalidAction = errors.New("invalid action")
)

// NewStrategy constructs a Strategy from strategy. strategy consists of a series of rules separated by '|'. Each rule
// is formatted as '<trigger>-<action>-|', rules must end with '-|'. An error is returned if strategy is not a valid
// strategy or is formatted incorrectly.
func NewStrategy(strategy string) (Strategy, error) {
	var rules []Rule

	// Split the string into rules, which are separated by '|', and parse each rule.
	parts := strings.SplitAfter(strategy, "|")
	switch {
	case parts[len(parts)-1] != "":
		return Strategy{}, fmt.Errorf("%w: %s, rules must end with '-|'", ErrInvalidRule, strategy)
	case parts[0] == "":
		return Strategy{}, errors.New("no rules found")
	default:
	}

	// The last element will be empty since each rule always ends with '|', so we ignore it.
	for _, rule := range parts[:len(parts)-1] {
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
func (s *Strategy) Apply(req *Request) {
	// iterate over each rule and if the trigger matches, apply the action tree to the target field.
	for _, rule := range s.Rules {
		if field, match := rule.Trigger.Match(req); match {
			// apply the action tree to the target field.
			// since the duplicate action can cause the tree to branch, the modifications are returned as a slice of
			// Fields which need to be applied to the request.
			mods := rule.Apply(field)
			// apply the modifications to the request.
			applyModifications(req, field, mods)
		}
	}
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
	return fmt.Sprintf("%s-%s-|", r.Trigger.String(), r.Tree.String())
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
func (t *Trigger) Match(req *Request) (Field, bool) {
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
		header := req.getHeader(t.TargetField)
		if header == "" {
			return Field{}, false
		}

		parts := strings.Split(header, ":")
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

// parseRule parses a string, rule, and returns a Rule. It returns an error if rule is not a valid rule or is
// formatted incorrectly.
func parseRule(r string) (Rule, error) {
	parts := strings.Split(r, "-")

	if len(parts) != 3 && parts[len(parts)-1] != "|" {
		return Rule{}, fmt.Errorf("%w: %s, should be formatted as '<trigger>-<actions>-|'", ErrInvalidRule, r)
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
			fmt.Errorf("%w: %s, trigger should be formatted as '[<proto>:<field>:<matchstr>]'", ErrInvalidRule, trigger)
	}

	proto := strings.ToUpper(parts[0][1:])
	switch proto {
	case "HTTP":
	case "DNS", "DNSQR":
		return Trigger{}, fmt.Errorf("%w: trigger protocols DNS and DNSQR are not supported yet", ErrInvalidRule)
	default:
		return Trigger{}, fmt.Errorf("%w: unsupported trigger protocol %q", ErrInvalidRule, proto)
	}

	field := strings.ToLower(parts[1])
	matchstr := strings.ToLower(parts[2][:len(parts[2])-1])

	return Trigger{
		Proto:       proto,
		TargetField: field,
		MatchStr:    matchstr,
	}, nil
}

// parseAction parses an action string in Geneva syntax and returns an Action. It returns an error if action is not a valid action or
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
		return nil, fmt.Errorf("%w: %s, missing matching parentheses", ErrInvalidRule, action)
	}

	// if we didn't find any parentheses, then there is no next action, so just construct the action.
	if fp == -1 {
		a, err := NewAction(action, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: %s, %s", ErrInvalidAction, action, err)
		}

		return a, nil
	}

	// there is a next action, so we need to split what's inside the parentheses into the left and right actions.
	l, r, err := splitLeftRight(action[fp : lp+1])
	if err != nil {
		return nil, err
	}

	// parse the left and right actions.
	var left, right, a Action
	if left, err = parseAction(l); err != nil {
		return nil, err
	}

	if right, err = parseAction(r); err != nil {
		return nil, err
	}

	// construct the action
	if a, err = NewAction(action[:fp], left, right); err != nil {
		return nil, fmt.Errorf("%w: %s, %s", ErrInvalidAction, action, err)
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
	return "", "", fmt.Errorf("%w: invalid format for left and right actions from %s", ErrInvalidRule, action)
}

// applyModifications applies the modifications, mods, to the field in the request. field is the original unmodified
// field.
func applyModifications(req *Request, field Field, mods []Field) {
	// iterate over mods and construct the new value.
	var newValue string
	if field.IsHeader {
		var vals []string
		for _, mod := range mods {
			vals = append(vals, mod.Name+":"+mod.Value)
		}

		newValue = strings.Join(vals, "\r\n")
	} else {
		for _, mod := range mods {
			newValue += mod.Value
		}
	}

	switch field.Name {
	case "method":
		req.method = newValue
	case "path":
		req.path = newValue
	case "version":
		req.version = newValue
	default:
		h := field.Name + ":" + field.Value
		req.headers = strings.Replace(req.headers, h, newValue, 1)
	}
}
