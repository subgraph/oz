package simplifier

import "github.com/twtiger/gosecco/tree"

// AcceptComparison implements Visitor
func (s *ltExpressionsSimplifier) AcceptComparison(a tree.Comparison) {
	l := s.Transform(a.Left)
	r := s.Transform(a.Right)

	newOp := a.Op

	switch a.Op {
	case tree.LT:
		newOp = tree.GTE
		l, r = r, l
	case tree.LTE:
		newOp = tree.GT
		l, r = r, l
	}

	s.Result = tree.Comparison{Op: newOp, Left: l, Right: r}

}

// ltExpressionsSimplifier simplifies LT and LTE expressions by rewriting them to GT and GTE expressions
type ltExpressionsSimplifier struct {
	tree.EmptyTransformer
}

func createLtExpressionsSimplifier() tree.Transformer {
	s := &ltExpressionsSimplifier{}
	s.RealSelf = s
	return s
}
