package simplifier

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type InclusionRemoverSimplifierSuite struct{}

var _ = Suite(&InclusionRemoverSimplifierSuite{})

func (s *InclusionRemoverSimplifierSuite) Test_removesInclusionStatementCorrectly(c *C) {
	sx := createInclusionRemoverSimplifier().Transform(
		tree.Inclusion{
			Positive: true,
			Left:     tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			Rights: []tree.Numeric{
				tree.NumericLiteral{1},
				tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
				tree.Arithmetic{Op: tree.MINUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (eq (plus 42 1) 1) (or (eq (plus 42 1) (plus 42 2)) (eq (plus 42 1) (minus 42 1))))")

}

func (s *InclusionRemoverSimplifierSuite) Test_removesNotInclusionStatementCorrectly(c *C) {
	sx := createInclusionRemoverSimplifier().Transform(
		tree.Inclusion{
			Positive: false,
			Left:     tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			Rights: []tree.Numeric{
				tree.NumericLiteral{1},
				tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
				tree.Arithmetic{Op: tree.MINUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			},
		},
	)
	//	c.Assert(tree.ExpressionString(sx), Equals, "(notIn (plus 42 1) 1 (plus 42 1) (minus 42 1))")
	c.Assert(tree.ExpressionString(sx), Equals, "(and (neq (plus 42 1) 1) (and (neq (plus 42 1) (plus 42 2)) (neq (plus 42 1) (minus 42 1))))")
}
