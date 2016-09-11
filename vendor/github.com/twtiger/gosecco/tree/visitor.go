package tree

// Visitor is a visitor for all parse nodes
type Visitor interface {
	AcceptAnd(And)
	AcceptArgument(Argument)
	AcceptArithmetic(Arithmetic)
	AcceptBinaryNegation(BinaryNegation)
	AcceptBooleanLiteral(BooleanLiteral)
	AcceptCall(Call)
	AcceptComparison(Comparison)
	AcceptInclusion(Inclusion)
	AcceptNegation(Negation)
	AcceptNumericLiteral(NumericLiteral)
	AcceptOr(Or)
	AcceptVariable(Variable)
}

// Expression is an AST expression
type Expression interface {
	Accept(Visitor)
}
