package tree

// Or represents an alternative
type Or struct {
	Left, Right Boolean
}

// Accept implements Expression
func (v Or) Accept(vs Visitor) {
	vs.AcceptOr(v)
}

// And represents an conjunction
type And struct {
	Left, Right Boolean
}

// Accept implements Expression
func (v And) Accept(vs Visitor) {
	vs.AcceptAnd(v)
}

// Negation represents a negation
type Negation struct {
	Operand Boolean
}

// Accept implements Expression
func (v Negation) Accept(vs Visitor) {
	vs.AcceptNegation(v)
}
