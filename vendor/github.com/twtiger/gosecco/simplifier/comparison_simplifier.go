package simplifier

import "github.com/twtiger/gosecco/tree"

// AcceptComparison implements Visitor
func (s *comparisonSimplifier) AcceptComparison(a tree.Comparison) {
	l := s.Transform(a.Left)
	r := s.Transform(a.Right)

	pl, ok1 := potentialExtractValue(l)
	pr, ok2 := potentialExtractValue(r)

	if ok1 && ok2 {
		switch a.Op {
		case tree.EQL:
			s.Result = tree.BooleanLiteral{pl == pr}
			return
		case tree.NEQL:
			s.Result = tree.BooleanLiteral{pl != pr}
			return
		case tree.GT:
			s.Result = tree.BooleanLiteral{pl > pr}
			return
		case tree.GTE:
			s.Result = tree.BooleanLiteral{pl >= pr}
			return
		case tree.LT:
			s.Result = tree.BooleanLiteral{pl < pr}
			return
		case tree.LTE:
			s.Result = tree.BooleanLiteral{pl <= pr}
			return
		case tree.BITSET:
			s.Result = tree.BooleanLiteral{pl&pr != 0}
			return
		}
	}
	s.Result = tree.Comparison{Op: a.Op, Left: l, Right: r}
}

// comparisonSimplifier simplifies comparison expressions by calculating them as much as possible
type comparisonSimplifier struct {
	tree.EmptyTransformer
}

func createComparisonSimplifier() tree.Transformer {
	s := &comparisonSimplifier{}
	s.RealSelf = s
	return s
}
