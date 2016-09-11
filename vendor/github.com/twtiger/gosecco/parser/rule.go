package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/twtiger/gosecco/tree"
)

var ruleHeadRE = regexp.MustCompile(`^[[:space:]]*([[:word:]]+)[[:space:]]*(?:\[(.*)\])?[[:space:]]*$`)

func findPositiveAndNegative(ss []string) (string, string, bool) {
	neg, pos := "", ""
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			if strings.HasPrefix(s, "+") {
				if pos != "" {
					return "", "", false
				}
				pos = strings.TrimPrefix(s, "+")
			} else if strings.HasPrefix(s, "-") {
				if neg != "" {
					return "", "", false
				}
				neg = strings.TrimPrefix(s, "-")
			} else {
				return "", "", false
			}
		}
	}
	return pos, neg, true
}

func parseRuleHead(s string) (tree.Rule, bool) {
	match := ruleHeadRE.FindStringSubmatch(s)
	if match != nil {
		positive, negative, ok := findPositiveAndNegative(strings.Split(match[2], ","))
		return tree.Rule{Name: match[1], PositiveAction: positive, NegativeAction: negative}, ok
	}
	return tree.Rule{}, false
}

func parseRule(s string) (tree.Rule, error) {
	parts := strings.SplitN(s, ":", 2) //This shouldn't fail since we will never hit this case unless linetype told us to
	rule, ok := parseRuleHead(parts[0])
	if !ok {
		return tree.Rule{}, errors.New("Invalid specification of syscall name")
	}

	if len(parts) < 2 || len(strings.TrimSpace(parts[1])) == 0 {
		return tree.Rule{}, fmt.Errorf("No expression specified for rule: %s", strings.TrimSpace(parts[0]))
	}

	x, hasReturn, ret, err := parseExpression(parts[1])
	if err != nil {
		return tree.Rule{}, err
	}
	if hasReturn {
		rule.PositiveAction = fmt.Sprintf("%d", ret)
	}
	rule.Body = x
	return rule, nil
}
