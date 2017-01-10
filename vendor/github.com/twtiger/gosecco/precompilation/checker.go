package precompilation

import (
	"fmt"

	"github.com/twtiger/gosecco/tree"
)

// The precompilation checker ensures that the given rule matches the
// language the compiler can process. This language is highly reduced and
// depends on the prior stages having been created
// Specifically, these rules are:
// - No full arguments
// - No literals larger than uint32
// - No calls
// - No variables
// - No in statements
// - No notIn statements
// - No LT
// - No LTE

type precompilationChecker struct {
	rules []*tree.Rule
}

// EnsureValid takes a policy and returns all the errors encounterered that is a mismatch for the compiler
// The kind of errors that this function generates will in general be because of programmer errors.
// If everything is valid, the return will be empty
func EnsureValid(p tree.Policy) []error {
	v := &precompilationChecker{rules: p.Rules}
	return v.check()
}

type ruleError struct {
	syscallName string
	err         error
}

func (e *ruleError) Error() string {
	return fmt.Sprintf("[%s] %s", e.syscallName, e.err)
}

func (v *precompilationChecker) check() []error {
	result := []error{}

	for _, r := range v.rules {
		if err := v.checkRule(r); err != nil {
			result = append(result, &ruleError{syscallName: r.Name, err: err})
		}
	}

	return result
}

func (v *precompilationChecker) checkRule(r *tree.Rule) error {
	return checkPrecompilationRules(r.Body)
}

func either(e1, e2 error) error {
	if e1 != nil {
		return e1
	}
	return e2
}
