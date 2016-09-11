package precompilation

import (
	"fmt"

	"github.com/twtiger/gosecco/tree"
)

type precompilationTypeChecker struct {
	result error
}

func checkPrecompilationRules(x tree.Expression) error {
	tc := &precompilationTypeChecker{result: nil}
	x.Accept(tc)
	return tc.result
}

// AcceptAnd implements Visitor
func (t *precompilationTypeChecker) AcceptAnd(v tree.And) {
	res := either(
		checkPrecompilationRules(v.Left),
		checkPrecompilationRules(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptArgument implements Visitor
func (t *precompilationTypeChecker) AcceptArgument(v tree.Argument) {
	if v.Type == tree.Full {
		t.result = fmt.Errorf("no full arguments allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
	}
}

// AcceptArithmetic implements Visitor
func (t *precompilationTypeChecker) AcceptArithmetic(v tree.Arithmetic) {
	res := either(
		checkPrecompilationRules(v.Left),
		checkPrecompilationRules(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptBinaryNegation implements Visitor
func (t *precompilationTypeChecker) AcceptBinaryNegation(v tree.BinaryNegation) {
	t.result = fmt.Errorf("no binary negation expressions allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
}

// AcceptBooleanLiteral implements Visitor
func (t *precompilationTypeChecker) AcceptBooleanLiteral(v tree.BooleanLiteral) {
}

// AcceptCall implements Visitor
func (t *precompilationTypeChecker) AcceptCall(v tree.Call) {
	t.result = fmt.Errorf("no calls allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
}

// AcceptComparison implements Visitor
func (t *precompilationTypeChecker) AcceptComparison(v tree.Comparison) {
	if v.Op == tree.LT {
		t.result = fmt.Errorf("no less than comparisons allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
	} else if v.Op == tree.LTE {
		t.result = fmt.Errorf("no less than or equals comparisons allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
	} else {
		res := either(
			checkPrecompilationRules(v.Left),
			checkPrecompilationRules(v.Right))
		if res != nil {
			t.result = res
		}
	}
}

// AcceptInclusion implements Visitor
func (t *precompilationTypeChecker) AcceptInclusion(v tree.Inclusion) {
	if v.Positive {
		t.result = fmt.Errorf("no inclusion expressions allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
	} else {
		t.result = fmt.Errorf("no negative inclusion expressions allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
	}
}

// AcceptNegation implements Visitor
func (t *precompilationTypeChecker) AcceptNegation(v tree.Negation) {
	if res := checkPrecompilationRules(v.Operand); res != nil {
		t.result = res
	}
}

// AcceptNumericLiteral implements Visitor
func (t *precompilationTypeChecker) AcceptNumericLiteral(v tree.NumericLiteral) {
	if v.Value > 0xFFFFFFFF {
		t.result = fmt.Errorf("no literals larger than 0xFFFFFFFF allowed - this is probably a programmer error: 0x%X", v.Value)
	}
}

// AcceptOr implements Visitor
func (t *precompilationTypeChecker) AcceptOr(v tree.Or) {
	res := either(
		checkPrecompilationRules(v.Left),
		checkPrecompilationRules(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptVariable implements Visitor
func (t *precompilationTypeChecker) AcceptVariable(v tree.Variable) {
	t.result = fmt.Errorf("no variables allowed - this is probably a programmer error: %s", v.Name)
}
