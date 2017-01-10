package tree

import "fmt"

// ExpressionString returns a string for the given expression
func ExpressionString(e Expression) string {
	sv := &StringVisitor{}
	e.Accept(sv)
	return sv.String()
}

// StringVisitor will generate an unambigious representation of an expression
type StringVisitor struct {
	result string
}

// String returns the current string built up
func (sv *StringVisitor) String() string {
	return sv.result
}

// AcceptAnd implements Visitor
func (sv *StringVisitor) AcceptAnd(v And) {
	sv.result = fmt.Sprintf("%s(and ", sv.result)
	v.Left.Accept(sv)
	sv.result += " "
	v.Right.Accept(sv)
	sv.result += ")"
}

// AcceptArgument implements Visitor
func (sv *StringVisitor) AcceptArgument(v Argument) {
	addition := ""
	if v.Type != Full {
		if v.Type == Hi {
			addition = "H"
		} else {
			addition = "L"
		}
	}
	sv.result += fmt.Sprintf("arg%s%d", addition, v.Index)
}

// AcceptArithmetic implements Visitor
func (sv *StringVisitor) AcceptArithmetic(v Arithmetic) {
	sv.result = fmt.Sprintf("%s(%s ", sv.result, ArithmeticSymbols[v.Op])
	v.Left.Accept(sv)
	sv.result += " "
	v.Right.Accept(sv)
	sv.result += ")"
}

// AcceptBinaryNegation implements Visitor
func (sv *StringVisitor) AcceptBinaryNegation(v BinaryNegation) {
	sv.result += "(binNeg "
	v.Operand.Accept(sv)
	sv.result += ")"
}

// AcceptBooleanLiteral implements Visitor
func (sv *StringVisitor) AcceptBooleanLiteral(v BooleanLiteral) {
	if v.Value {
		sv.result += "true"
	} else {
		sv.result += "false"
	}
}

// AcceptCall implements Visitor
func (sv *StringVisitor) AcceptCall(v Call) {
	sv.result += "(" + v.Name
	for _, a := range v.Args {
		sv.result += " "
		a.Accept(sv)
	}
	sv.result += ")"
}

// AcceptComparison implements Visitor
func (sv *StringVisitor) AcceptComparison(v Comparison) {
	sv.result = fmt.Sprintf("%s(%s ", sv.result, ComparisonSymbols[v.Op])
	v.Left.Accept(sv)
	sv.result += " "
	v.Right.Accept(sv)
	sv.result += ")"
}

// AcceptInclusion implements Visitor
func (sv *StringVisitor) AcceptInclusion(v Inclusion) {
	name := "in"
	if !v.Positive {
		name = "notIn"
	}
	sv.result += "(" + name + " "
	v.Left.Accept(sv)
	sep := " "
	for _, a := range v.Rights {
		sv.result += sep
		a.Accept(sv)
	}
	sv.result += ")"
}

// AcceptNegation implements Visitor
func (sv *StringVisitor) AcceptNegation(v Negation) {
	sv.result += "(not "
	v.Operand.Accept(sv)
	sv.result += ")"
}

// AcceptNumericLiteral implements Visitor
func (sv *StringVisitor) AcceptNumericLiteral(v NumericLiteral) {
	sv.result += fmt.Sprintf("%d", v.Value)
}

// AcceptOr implements Visitor
func (sv *StringVisitor) AcceptOr(v Or) {
	sv.result = fmt.Sprintf("%s(or ", sv.result)
	v.Left.Accept(sv)
	sv.result += " "
	v.Right.Accept(sv)
	sv.result += ")"
}

// AcceptVariable implements Visitor
func (sv *StringVisitor) AcceptVariable(v Variable) {
	sv.result += v.Name
}
