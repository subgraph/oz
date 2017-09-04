package simplifier

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type LtExpressionsSimplifierSuite struct{}

var _ = Suite(&LtExpressionsSimplifierSuite{})

func (s *LtExpressionsSimplifierSuite) Test_simplifyLTExpression(c *C) {
	sx := createLtExpressionsSimplifier().Transform(tree.Comparison{Op: tree.LT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})

	c.Assert(tree.ExpressionString(sx), Equals, "(gte 15 42)")
}

func (s *LtExpressionsSimplifierSuite) Test_simplifyLTEExpression(c *C) {
	sx := createLtExpressionsSimplifier().Transform(tree.Comparison{Op: tree.LTE, Left: tree.NumericLiteral{43}, Right: tree.NumericLiteral{16}})

	c.Assert(tree.ExpressionString(sx), Equals, "(gt 16 43)")
}
