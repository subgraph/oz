package tree

// Inclusion represents either a positive or a negative inclusion operation
type Inclusion struct {
	Positive bool
	Left     Numeric
	Rights   []Numeric
}

// Accept implements Expression
func (v Inclusion) Accept(vs Visitor) {
	vs.AcceptInclusion(v)
}
