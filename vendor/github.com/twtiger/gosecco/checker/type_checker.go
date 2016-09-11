package checker

import (
	"fmt"

	"github.com/twtiger/gosecco/tree"
)

type typeChecker struct {
	result        error
	expectBoolean bool
}

func typeCheckExpectingBoolean(x tree.Expression) error {
	tc := &typeChecker{result: nil, expectBoolean: true}
	x.Accept(tc)
	return tc.result
}

func typeCheckExpectingNumeric(x tree.Expression) error {
	tc := &typeChecker{result: nil, expectBoolean: false}
	x.Accept(tc)
	return tc.result
}

// AcceptAnd implements Visitor
func (t *typeChecker) AcceptAnd(v tree.And) {
	if !t.expectBoolean {
		t.result = fmt.Errorf("expected numeric expression but found: %s", tree.ExpressionString(v))
		return
	}

	res := either(
		typeCheckExpectingBoolean(v.Left),
		typeCheckExpectingBoolean(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptArgument implements Visitor
func (t *typeChecker) AcceptArgument(v tree.Argument) {
	if t.expectBoolean {
		t.result = fmt.Errorf("expected boolean expression but found: %s", tree.ExpressionString(v))
	}
}

// AcceptArithmetic implements Visitor
func (t *typeChecker) AcceptArithmetic(v tree.Arithmetic) {
	if t.expectBoolean {
		t.result = fmt.Errorf("expected boolean expression but found: %s", tree.ExpressionString(v))
		return
	}

	res := either(
		typeCheckExpectingNumeric(v.Left),
		typeCheckExpectingNumeric(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptBinaryNegation implements Visitor
func (t *typeChecker) AcceptBinaryNegation(v tree.BinaryNegation) {
	if t.expectBoolean {
		t.result = fmt.Errorf("expected boolean expression but found: %s", tree.ExpressionString(v))
		return
	}

	res := typeCheckExpectingNumeric(v.Operand)
	if res != nil {
		t.result = res
	}
}

// AcceptBooleanLiteral implements Visitor
func (t *typeChecker) AcceptBooleanLiteral(v tree.BooleanLiteral) {
	if !t.expectBoolean {
		t.result = fmt.Errorf("expected numeric expression but found: %s", tree.ExpressionString(v))
	}
}

// AcceptCall implements Visitor
func (t *typeChecker) AcceptCall(v tree.Call) {
	t.result = fmt.Errorf("found unresolved call: %s", v.Name)
}

// AcceptComparison implements Visitor
func (t *typeChecker) AcceptComparison(v tree.Comparison) {
	if !t.expectBoolean {
		t.result = fmt.Errorf("expected numeric expression but found: %s", tree.ExpressionString(v))
		return
	}

	// This language only accepts comparisons between numeric values, so we implement that here
	res := either(
		typeCheckExpectingNumeric(v.Left),
		typeCheckExpectingNumeric(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptInclusion implements Visitor
func (t *typeChecker) AcceptInclusion(v tree.Inclusion) {
	if !t.expectBoolean {
		t.result = fmt.Errorf("expected numeric expression but found: %s", tree.ExpressionString(v))
		return
	}

	res := typeCheckExpectingNumeric(v.Left)
	for _, r := range v.Rights {
		res2 := typeCheckExpectingNumeric(r)
		if res == nil {
			res = res2
		}
	}

	if res != nil {
		t.result = res
	}

}

// AcceptNegation implements Visitor
func (t *typeChecker) AcceptNegation(v tree.Negation) {
	if !t.expectBoolean {
		t.result = fmt.Errorf("expected numeric expression but found: %s", tree.ExpressionString(v))
		return
	}

	res := typeCheckExpectingBoolean(v.Operand)
	if res != nil {
		t.result = res
	}
}

// AcceptNumericLiteral implements Visitor
func (t *typeChecker) AcceptNumericLiteral(v tree.NumericLiteral) {
	if t.expectBoolean {
		t.result = fmt.Errorf("expected boolean expression but found: %s", tree.ExpressionString(v))
	}
}

// AcceptOr implements Visitor
func (t *typeChecker) AcceptOr(v tree.Or) {
	if !t.expectBoolean {
		t.result = fmt.Errorf("expected numeric expression but found: %s", tree.ExpressionString(v))
		return
	}

	res := either(
		typeCheckExpectingBoolean(v.Left),
		typeCheckExpectingBoolean(v.Right))
	if res != nil {
		t.result = res
	}
}

// AcceptVariable implements Visitor
func (t *typeChecker) AcceptVariable(v tree.Variable) {
	t.result = fmt.Errorf("found unresolved variable: %s", v.Name)
}
