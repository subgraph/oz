package tree

// Boolean represents a boolean expression
type Boolean interface {
	Expression
}

// Numeric represents a numeric expression
type Numeric interface {
	Expression
}

// Any represents either a boolean or numeric expression
type Any interface {
	Expression
	// IsBoolean() bool
	// AsBoolean() Boolean
	// IsNumeric() bool
	// AsNumeric() Numeric
}
