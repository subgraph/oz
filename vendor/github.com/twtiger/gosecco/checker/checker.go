package checker

import (
	"errors"
	"fmt"

	"github.com/twtiger/gosecco/constants"
	"github.com/twtiger/gosecco/tree"
)

// The assumption is that the input to the checker is a simplified, unified
// policy that is ready to be compiled. The checker does the final step of making sure that
// all the rules are valid and type checks.
// The checker will not do anything with the macros defined.
// It will assume all calls and variable references left are errors (but that should have been caught
// in the phases before).
// Except for checking type validity, the checker will also make sure we don't have
// more than one rule for the same syscall. This is also the place where we make sure
// all the syscalls with rules are defined.
// Further, we will also ensure that the usage of Arguments matches the behavior we are interested in
// Specifically, full Arguments can only appear directly on the side of comparisons, never inside
// arithmetic expressions.

// EnsureValid takes a policy and returns all the errors encounterered for the given rules
// If everything is valid, the return will be empty
func EnsureValid(p tree.Policy) []error {
	v := &validityChecker{rules: p.Rules, seen: make(map[string]*tree.Rule)}
	return v.check()
}

type validityChecker struct {
	rules []*tree.Rule
	seen  map[string]*tree.Rule
}

type ruleError struct {
	syscallName string
	err         error
}

func (e *ruleError) Error() string {
	return fmt.Sprintf("[%s] %s", e.syscallName, e.err)
}

func checkValidSyscall(r *tree.Rule) error {
	if _, ok := constants.GetSyscall(r.Name); !ok {
		return errors.New("invalid syscall")
	}
	return nil
}

func (v *validityChecker) check() []error {
	result := []error{}

	for _, r := range v.rules {
		var res error
		oldR, ok := v.seen[r.Name]
		if ok && (r.PositiveAction != oldR.PositiveAction ||
			r.NegativeAction != oldR.NegativeAction ||
			r.Body != oldR.Body) {
			res = errors.New("duplicate definition of syscall rule")
		}
		v.seen[r.Name] = r
		if res == nil {
			res = checkValidSyscall(r)
		}
		if res == nil {
			res = v.checkRule(r)
		}
		if res != nil {
			result = append(result, &ruleError{syscallName: r.Name, err: res})
		}
	}

	return result
}

func either(e1, e2 error) error {
	if e1 != nil {
		return e1
	}
	return e2
}

func (v *validityChecker) checkRule(r *tree.Rule) error {
	return either(
		typeCheckExpectingBoolean(r.Body),
		checkRestrictedArgumentUsage(r.Body))

}
