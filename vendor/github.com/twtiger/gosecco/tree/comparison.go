package tree

// ComparisonType specifies the possible comparison types
type ComparisonType int

// Contains all the comparison types
const (
	EQL ComparisonType = iota
	NEQL
	GT
	GTE
	LT
	LTE
	BITSET
)

// ComparisonNames maps types to names for presentation
var ComparisonNames = map[ComparisonType]string{
	EQL:    "==",
	NEQL:   "!=",
	GT:     ">",
	GTE:    ">=",
	LT:     "<",
	LTE:    "<=",
	BITSET: "&?",
}

// ComparisonSymbols maps types to names for symbolic processing
var ComparisonSymbols = map[ComparisonType]string{
	EQL:    "eq",
	NEQL:   "neq",
	GT:     "gt",
	GTE:    "gte",
	LT:     "lt",
	LTE:    "lte",
	BITSET: "bitset",
}

// Comparison represents a comparison
type Comparison struct {
	Op          ComparisonType
	Left, Right Numeric
}

// Accept implements Expression
func (v Comparison) Accept(vs Visitor) {
	vs.AcceptComparison(v)
}
