package tree

// BooleanLiteral represents a boolean literal
type BooleanLiteral struct {
	Value bool
}

// Accept implements Expression
func (v BooleanLiteral) Accept(vs Visitor) {
	vs.AcceptBooleanLiteral(v)
}

// NumericLiteral represents a numeric literal
type NumericLiteral struct {
	Value uint64
}

// Accept implements Expression
func (v NumericLiteral) Accept(vs Visitor) {
	vs.AcceptNumericLiteral(v)
}
