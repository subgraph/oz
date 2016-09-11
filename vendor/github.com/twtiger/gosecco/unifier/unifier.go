package unifier

import (
	"strconv"

	"github.com/twtiger/gosecco/tree"
)

func getDefaultAction(t tree.Macro) string {
	switch f := t.Body.(type) {
	case tree.NumericLiteral:
		return strconv.Itoa(int(f.Value))
	case tree.Variable:
		return f.Name
	}
	panic("shouldn't happen")
}

func addAllToMap(to, from map[string]tree.Macro) {
	for k, v := range from {
		to[k] = v
	}
}

func combineMacroMaps(ms []map[string]tree.Macro) map[string]tree.Macro {
	result := make(map[string]tree.Macro)

	for _, mm := range ms {
		addAllToMap(result, mm)
	}

	return result
}

// Unify will unify all variables and calls in the given rule set with the macros in the same file. The macros in the same file will
// be evaluated linearly, so it is possible to use the same variable name multiple times. The additionalMacros provide access to
// variables defined in other files. The list of additional macros will be combined in such a way that the names in later maps override
// the names in the earlier maps. The default positive and negative actions can be overridden in the files by providing DEFAULT_POSITIVE
// and DEFAULT_NEGATIVE variables anywhere in the files. The default actions can only be defined once in a file, and will be in effect
// for all rules in that file, unless a specific rule overrides the default actions.
func Unify(r tree.RawPolicy, additionalMacros []map[string]tree.Macro, defaultPositive, defaultNegative, defaultPolicy string) (tree.Policy, error) {
	var rules []*tree.Rule
	macros := combineMacroMaps(additionalMacros)
	collectedMacros := make(map[string]tree.Macro)
	for _, e := range r.RuleOrMacros {
		switch v := e.(type) {
		case tree.Rule:
			r, err := replaceFreeNames(v, macros)
			if err != nil {
				return tree.Policy{}, err
			}
			rules = append(rules, &r)
		case tree.Macro:
			switch v.Name {
			case "DEFAULT_POSITIVE":
				defaultPositive = getDefaultAction(v)
			case "DEFAULT_NEGATIVE":
				defaultNegative = getDefaultAction(v)
			case "DEFAULT_POLICY":
				defaultPolicy = getDefaultAction(v)
			default:
				macros[v.Name] = v
				collectedMacros[v.Name] = v
			}
		}
	}
	return tree.Policy{DefaultPositiveAction: defaultPositive, DefaultNegativeAction: defaultNegative, DefaultPolicyAction: defaultPolicy, Macros: collectedMacros, Rules: rules}, nil
}

func replaceFreeNames(r tree.Rule, macros map[string]tree.Macro) (tree.Rule, error) {
	body, err := replace(r.Body, macros)
	rule := tree.Rule{
		Name:           r.Name,
		PositiveAction: r.PositiveAction,
		NegativeAction: r.NegativeAction,
		Body:           body,
	}
	return rule, err
}

func replace(x tree.Expression, macros map[string]tree.Macro) (tree.Expression, error) {
	r := &replacer{expression: x, macros: macros, err: nil}
	x.Accept(r)
	if r.err != nil {
		return nil, r.err
	}
	return r.expression, nil
}
