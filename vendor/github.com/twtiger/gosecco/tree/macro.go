package tree

// Macro represents either a simple variable or a more complicated macro/func expression
type Macro struct {
	Name          string
	ArgumentNames []string
	Body          Expression
}
