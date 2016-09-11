package tree

// Call represents an application of a method/macro
type Call struct {
	Name string
	Args []Any
}

// Accept implements Expression
func (v Call) Accept(vs Visitor) {
	vs.AcceptCall(v)
}
