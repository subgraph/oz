package parser

import (
	"errors"
	"regexp"
	"strings"

	"github.com/twtiger/gosecco/tree"
)

var parseBindingHeadRE = regexp.MustCompile(`^[[:space:]]*([[:word:]]+)[[:space:]]*(?:\((.*)\))?[[:space:]]*$`)

func parseArgumentNames(s string) []string {
	ss := strings.Split(s, ",")
	result := make([]string, len(ss))

	for ix, sss := range ss {
		result[ix] = strings.TrimSpace(sss)
	}

	return result
}

func parseBindingHead(s string) (tree.Macro, bool) {
	match := parseBindingHeadRE.FindStringSubmatch(s)
	if match != nil {
		m := tree.Macro{
			Name: match[1],
		}
		if len(match) > 2 {
			bla := strings.TrimSpace(match[2])
			if len(bla) > 0 {
				m.ArgumentNames = parseArgumentNames(bla)
			}
		}
		return m, true
	}
	return tree.Macro{}, false
}

func parseBinding(s string) (tree.Macro, error) {
	parts := strings.SplitN(s, "=", 2) //This shouldn't fail since we will never hit this case unless linetype told us to
	binding, ok := parseBindingHead(parts[0])
	if !ok {
		return tree.Macro{}, errors.New("Invalid macro name")
	}

	x, _, _, err := parseExpressionForBinding(parts[1])
	if err != nil {
		return tree.Macro{}, err
	}
	binding.Body = x
	return binding, nil
}
