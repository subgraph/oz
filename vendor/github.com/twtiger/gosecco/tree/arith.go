package tree

// ArithmeticType specifies the different possible arithmetic operations
type ArithmeticType int

// Constants for the different possible types
const (
	PLUS ArithmeticType = iota
	MINUS
	MULT
	DIV
	BINAND
	BINOR
	BINXOR
	LSH
	RSH
	MOD
)

// ArithmeticNames maps the types to names for presentation
var ArithmeticNames = map[ArithmeticType]string{
	PLUS:   "+",
	MINUS:  "-",
	MULT:   "*",
	DIV:    "/",
	BINAND: "&",
	BINOR:  "|",
	BINXOR: "^",
	LSH:    "<<",
	RSH:    ">>",
	MOD:    "%",
}

// ArithmeticSymbols maps the types to names for symbolic processing
var ArithmeticSymbols = map[ArithmeticType]string{
	PLUS:   "plus",
	MINUS:  "minus",
	MULT:   "mul",
	DIV:    "div",
	BINAND: "binand",
	BINOR:  "binor",
	BINXOR: "binxor",
	LSH:    "lsh",
	RSH:    "rsh",
	MOD:    "mod",
}

// Arithmetic represents an arithmetic operation
type Arithmetic struct {
	Op          ArithmeticType
	Left, Right Numeric
}

// Accept implements Expression
func (v Arithmetic) Accept(vs Visitor) {
	vs.AcceptArithmetic(v)
}

// BinaryNegation represents binary negation of a number
type BinaryNegation struct {
	Operand Numeric
}

// Accept implements Expression
func (v BinaryNegation) Accept(vs Visitor) {
	vs.AcceptBinaryNegation(v)
}
