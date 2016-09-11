package tree

// ArgumentType represents one of the three types of argument loads we can do
type ArgumentType int

// The different types of argument loads that can happen
const (
	Full ArgumentType = iota
	Low
	Hi
)

// Argument represents an argment given to the syscall
type Argument struct {
	Type  ArgumentType
	Index int
}

// Accept implements Expression
func (v Argument) Accept(vs Visitor) {
	vs.AcceptArgument(v)
}

// Variable represents a variable used before
type Variable struct {
	Name string
}

// Accept implements Expression
func (v Variable) Accept(vs Visitor) {
	vs.AcceptVariable(v)
}
