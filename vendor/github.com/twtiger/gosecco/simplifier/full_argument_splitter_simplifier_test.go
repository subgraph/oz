package simplifier

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type FullArgumentSplitterSimplifierSuite struct{}

var _ = Suite(&FullArgumentSplitterSimplifierSuite{})

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesEqualityWithArgAgainstNumber(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.EQL,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.NumericLiteral{0x123456789ABCDEF0},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq argL2 2596069104) (eq argH2 305419896))")

	sx = createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.EQL,
			Left:  tree.NumericLiteral{0x123456789ABCDEF0},
			Right: tree.Argument{Type: tree.Full, Index: 2},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq 2596069104 argL2) (eq 305419896 argH2))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesBitandWithArgAgainstNumber(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.BITSET,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.NumericLiteral{0x123456789ABCDEF0},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq (binand argL2 2596069104) 2596069104) (eq (binand argH2 305419896) 305419896))")

	sx = createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.BITSET,
			Left:  tree.NumericLiteral{0x123456789ABCDEF0},
			Right: tree.Argument{Type: tree.Full, Index: 2},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq (binand 2596069104 argL2) argL2) (eq (binand 305419896 argH2) argH2))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesEqualityWithArgAgainstArg(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.EQL,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.Argument{Type: tree.Full, Index: 4},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq argL2 argL4) (eq argH2 argH4))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesBitsetWithArgAgainstArg(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.BITSET,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.Argument{Type: tree.Full, Index: 4},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq (binand argL2 argL4) argL4) (eq (binand argH2 argH4) argH4))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesBitsetWithArgAgainstExpression(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.BITSET,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.Arithmetic{Op: tree.BINOR, Left: tree.NumericLiteral{0x1}, Right: tree.NumericLiteral{0x2}},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq argH2 0) (neq (binand argL2 (binor 1 2)) 0))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesNonequalityWithArgAgainstNumber(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.NEQL,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.NumericLiteral{0x123456789ABCDEF0},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (neq argL2 2596069104) (neq argH2 305419896))")

	sx = createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.NEQL,
			Left:  tree.NumericLiteral{0x123456789ABCDEF0},
			Right: tree.Argument{Type: tree.Full, Index: 2},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (neq 2596069104 argL2) (neq 305419896 argH2))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesNonequalityWithArgAgainstArg(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.NEQL,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.Argument{Type: tree.Full, Index: 1},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (neq argL2 argL1) (neq argH2 argH1))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesGtWithArgAgainstNumber(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.GT,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.NumericLiteral{0x123456789ABCDEF0},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (gt argH2 305419896) (and (eq argH2 305419896) (gt argL2 2596069104)))")

	sx = createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.GT,
			Left:  tree.NumericLiteral{0x123456789ABCDEF0},
			Right: tree.Argument{Type: tree.Full, Index: 2},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (gt 305419896 argH2) (and (eq 305419896 argH2) (gt 2596069104 argL2)))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesGtWithArgAgainstArg(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.GT,
			Left:  tree.Argument{Type: tree.Full, Index: 3},
			Right: tree.Argument{Type: tree.Full, Index: 0},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (gt argH3 argH0) (and (eq argH3 argH0) (gt argL3 argL0)))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesGteWithArgAgainstNumber(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.GTE,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.NumericLiteral{0x123456789ABCDEF0},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (gt argH2 305419896) (and (eq argH2 305419896) (gte argL2 2596069104)))")

	sx = createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.GTE,
			Left:  tree.NumericLiteral{0x123456789ABCDEF0},
			Right: tree.Argument{Type: tree.Full, Index: 2},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (gt 305419896 argH2) (and (eq 305419896 argH2) (gte 2596069104 argL2)))")
}

func (s *FullArgumentSplitterSimplifierSuite) Test_simplifiesGteWithArgAgainstArg(c *C) {
	sx := createFullArgumentSplitterSimplifier().Transform(
		tree.Comparison{
			Op:    tree.GTE,
			Left:  tree.Argument{Type: tree.Full, Index: 2},
			Right: tree.Argument{Type: tree.Full, Index: 5},
		},
	)

	c.Assert(tree.ExpressionString(sx), Equals, "(or (gt argH2 argH5) (and (eq argH2 argH5) (gte argL2 argL5)))")
}
