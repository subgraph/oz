package simplifier

import "github.com/twtiger/gosecco/tree"

// AcceptOr implements Visitor
func (s *booleanSimplifier) AcceptOr(a tree.Or) {
	l := s.Transform(a.Left)
	r := s.Transform(a.Right)
	pl, ok1 := potentialExtractBooleanValue(l)
	pr, ok2 := potentialExtractBooleanValue(r)
	// First branch is possible to calculate at compile time
	if ok1 {
		if pl {
			s.Result = tree.BooleanLiteral{true}
		} else {
			if ok2 {
				s.Result = tree.BooleanLiteral{pr}
			} else {
				s.Result = r
			}
		}
	} else {
		s.Result = tree.Or{l, r}
	}
}

// AcceptAnd implements Visitor
func (s *booleanSimplifier) AcceptAnd(a tree.And) {
	l := s.Transform(a.Left)
	r := s.Transform(a.Right)
	pl, ok1 := potentialExtractBooleanValue(l)
	pr, ok2 := potentialExtractBooleanValue(r)
	// First branch is possible to calculate at compile time
	if ok1 {
		if pl {
			// If the first branch is always true, we are determined by the second branch
			if ok2 {
				s.Result = tree.BooleanLiteral{pr}
			} else {
				s.Result = r
			}
		} else {
			// If the first branch is always false, we can never succeed
			s.Result = tree.BooleanLiteral{false}
		}
	} else {
		// Second branch is possible to calculate at compile time
		if ok2 {
			if pr {
				// If the second branch statically evaluates to true, the and expression is determined by the left arm
				s.Result = l
			} else {
				// And if the second branch is false, it doesn't matter what the first branch is
				s.Result = tree.BooleanLiteral{false}
			}
		} else {
			s.Result = tree.And{l, r}
		}
	}
}

// AcceptNegation implements Visitor
func (s *booleanSimplifier) AcceptNegation(v tree.Negation) {
	val := s.Transform(v.Operand)
	val2, ok := potentialExtractBooleanValue(val)
	if ok {
		s.Result = tree.BooleanLiteral{!val2}
	}
}

// booleanSimplifier simplifies boolean expressions by calculating them as much as possible
type booleanSimplifier struct {
	tree.EmptyTransformer
}

func createBooleanSimplifier() tree.Transformer {
	s := &booleanSimplifier{}
	s.RealSelf = s
	return s
}
