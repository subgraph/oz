package checker

import (
	"fmt"

	"github.com/twtiger/gosecco/tree"
)

type argumentRestrictions struct {
	result error
}

func (ar *argumentRestrictions) register(e error) {
	ar.result = either(ar.result, e)
}

func checkRestrictedArgumentUsage(x tree.Expression) error {
	ar := &argumentRestrictions{result: nil}
	x.Accept(ar)
	return ar.result
}

// AcceptAnd implements Visitor
func (ar *argumentRestrictions) AcceptAnd(v tree.And) {
	ar.register(either(
		checkRestrictedArgumentUsage(v.Left),
		checkRestrictedArgumentUsage(v.Right)))
}

// AcceptArgument implements Visitor
func (ar *argumentRestrictions) AcceptArgument(v tree.Argument) {
	if v.Type == tree.Full {
		ar.register(fmt.Errorf("full argument cannot be used in arithmetic expressions - use the 32bit accessors instead: %s", tree.ExpressionString(v)))
	}
}

// AcceptArithmetic implements Visitor
func (ar *argumentRestrictions) AcceptArithmetic(v tree.Arithmetic) {
	ar.register(either(
		checkRestrictedArgumentUsage(v.Left),
		checkRestrictedArgumentUsage(v.Right)))
}

// AcceptBinaryNegation implements Visitor
func (ar *argumentRestrictions) AcceptBinaryNegation(v tree.BinaryNegation) {
	ar.register(checkRestrictedArgumentUsage(v.Operand))
}

// AcceptBooleanLiteral implements Visitor
func (ar *argumentRestrictions) AcceptBooleanLiteral(v tree.BooleanLiteral) {
	// All good
}

// AcceptCall implements Visitor
func (ar *argumentRestrictions) AcceptCall(v tree.Call) {
	// Ignore - type checker will find this
}

// AcceptComparison implements Visitor
func (ar *argumentRestrictions) AcceptComparison(v tree.Comparison) {
	_, hasArgL := v.Left.(tree.Argument)
	_, hasArgR := v.Right.(tree.Argument)

	if !hasArgL {
		ar.register(checkRestrictedArgumentUsage(v.Left))
	}
	if !hasArgR {
		ar.register(checkRestrictedArgumentUsage(v.Right))
	}
}

// AcceptInclusion implements Visitor
func (ar *argumentRestrictions) AcceptInclusion(v tree.Inclusion) {
	ar.register(checkRestrictedArgumentUsage(v.Left))
	for _, r := range v.Rights {
		ar.register(checkRestrictedArgumentUsage(r))
	}
}

// AcceptNegation implements Visitor
func (ar *argumentRestrictions) AcceptNegation(v tree.Negation) {
	ar.register(checkRestrictedArgumentUsage(v.Operand))
}

// AcceptNumericLiteral implements Visitor
func (ar *argumentRestrictions) AcceptNumericLiteral(v tree.NumericLiteral) {
	// All good
}

// AcceptOr implements Visitor
func (ar *argumentRestrictions) AcceptOr(v tree.Or) {
	ar.register(either(
		checkRestrictedArgumentUsage(v.Left),
		checkRestrictedArgumentUsage(v.Right)))
}

// AcceptVariable implements Visitor
func (ar *argumentRestrictions) AcceptVariable(v tree.Variable) {
	// Ignore - type checker will find this
}
