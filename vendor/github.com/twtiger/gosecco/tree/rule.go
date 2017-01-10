package tree

// Rule contains all the information for one specific rule
type Rule struct {
	Name           string
	PositiveAction string
	NegativeAction string
	Body           Expression
}
