package simplifier

import "github.com/twtiger/gosecco/tree"

// AcceptBinaryNegation implements Visitor
func (s *arithmeticSimplifier) AcceptBinaryNegation(v tree.BinaryNegation) {
	val := s.Transform(v.Operand)
	if val2, ok := potentialExtractValue(val); ok {
		s.Result = tree.NumericLiteral{^val2}
	}
}

// AcceptArithmetic implements Visitor
func (s *arithmeticSimplifier) AcceptArithmetic(a tree.Arithmetic) {
	l := s.Transform(a.Left)
	r := s.Transform(a.Right)

	pl, ok1 := potentialExtractValue(l)
	pr, ok2 := potentialExtractValue(r)

	if ok1 && ok2 {
		switch a.Op {
		case tree.PLUS:
			s.Result = tree.NumericLiteral{pl + pr}
			return
		case tree.MINUS:
			s.Result = tree.NumericLiteral{pl - pr}
			return
		case tree.MULT:
			s.Result = tree.NumericLiteral{pl * pr}
			return
		case tree.DIV:
			s.Result = tree.NumericLiteral{pl / pr}
			return
		case tree.MOD:
			s.Result = tree.NumericLiteral{pl % pr}
			return
		case tree.BINAND:
			s.Result = tree.NumericLiteral{pl & pr}
			return
		case tree.BINOR:
			s.Result = tree.NumericLiteral{pl | pr}
			return
		case tree.BINXOR:
			s.Result = tree.NumericLiteral{pl ^ pr}
			return
		case tree.LSH:
			s.Result = tree.NumericLiteral{pl << pr}
			return
		case tree.RSH:
			s.Result = tree.NumericLiteral{pl >> pr}
			return
		}
	}
	s.Result = tree.Arithmetic{Op: a.Op, Left: l, Right: r}
}

// arithmeticSimplifier simplifies arithmetic expressions by calculating them as much as possible
type arithmeticSimplifier struct {
	tree.EmptyTransformer
}

func createArithmeticSimplifier() tree.Transformer {
	s := &arithmeticSimplifier{}
	s.RealSelf = s
	return s
}
