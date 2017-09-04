package simplifier

import (
	"testing"

	"github.com/twtiger/gosecco/tree"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SimplifierSuite struct{}

var _ = Suite(&SimplifierSuite{})

func (s *SimplifierSuite) Test_simplifyAddition(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{2}})
	c.Assert(tree.ExpressionString(sx), Equals, "3")
}

func (s *SimplifierSuite) Test_simplifySubtraction(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.MINUS, Left: tree.NumericLiteral{32}, Right: tree.NumericLiteral{3}})
	c.Assert(tree.ExpressionString(sx), Equals, "29")
}

func (s *SimplifierSuite) Test_simplifyMult(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{12}, Right: tree.NumericLiteral{3}})
	c.Assert(tree.ExpressionString(sx), Equals, "36")
}

func (s *SimplifierSuite) Test_simplifyDiv(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.DIV, Left: tree.NumericLiteral{37}, Right: tree.NumericLiteral{3}})
	c.Assert(tree.ExpressionString(sx), Equals, "12")
}

func (s *SimplifierSuite) Test_simplifyMod(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.MOD, Left: tree.NumericLiteral{37}, Right: tree.NumericLiteral{3}})
	c.Assert(tree.ExpressionString(sx), Equals, "1")
}

func (s *SimplifierSuite) Test_simplifyBinAnd(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.BINAND, Left: tree.NumericLiteral{7}, Right: tree.NumericLiteral{4}})
	c.Assert(tree.ExpressionString(sx), Equals, "4")
}

func (s *SimplifierSuite) Test_simplifyBinOr(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.BINOR, Left: tree.NumericLiteral{3}, Right: tree.NumericLiteral{8}})
	c.Assert(tree.ExpressionString(sx), Equals, "11")
}

func (s *SimplifierSuite) Test_simplifyBinXoe(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.BINXOR, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{12}})
	c.Assert(tree.ExpressionString(sx), Equals, "38")
}

func (s *SimplifierSuite) Test_simplifyLsh(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}})
	c.Assert(tree.ExpressionString(sx), Equals, "168")
}

func (s *SimplifierSuite) Test_simplifyRsh(c *C) {
	sx := Simplify(tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{84}, Right: tree.NumericLiteral{2}})
	c.Assert(tree.ExpressionString(sx), Equals, "21")
}

func (s *SimplifierSuite) Test_simplifyCall(c *C) {
	sx := Simplify(tree.Call{"foo", []tree.Any{tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{84}, Right: tree.NumericLiteral{2}}}})
	c.Assert(tree.ExpressionString(sx), Equals, "(foo 21)")
}

func (s *SimplifierSuite) Test_simplifyAnd(c *C) {
	sx := Simplify(tree.And{
		tree.Comparison{
			Left:  tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		},
		tree.Comparison{
			Left:  tree.Argument{Index: 2},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		},
	})
	c.Assert(tree.ExpressionString(sx), Equals, "false")

	sx = Simplify(tree.And{
		tree.Comparison{
			Left:  tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.Variable{"foo"}},
		},
		tree.Comparison{
			Left:  tree.Variable{"argxxx"},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		},
	})
	c.Assert(tree.ExpressionString(sx), Equals, "(and (eq 168 (rsh 42 foo)) (eq argxxx 10))")
}

func (s *SimplifierSuite) Test_simplifyComparison(c *C) {
	sx := Simplify(tree.Comparison{Left: tree.NumericLiteral{42}, Op: tree.EQL, Right: tree.NumericLiteral{41}})
	c.Assert(tree.ExpressionString(sx), Equals, "false")

	sx = Simplify(tree.Comparison{Left: tree.NumericLiteral{42}, Op: tree.NEQL, Right: tree.NumericLiteral{41}})
	c.Assert(tree.ExpressionString(sx), Equals, "true")

	sx = Simplify(tree.Comparison{Left: tree.NumericLiteral{42}, Op: tree.GT, Right: tree.NumericLiteral{41}})
	c.Assert(tree.ExpressionString(sx), Equals, "true")

	sx = Simplify(tree.Comparison{Left: tree.NumericLiteral{42}, Op: tree.GTE, Right: tree.NumericLiteral{41}})
	c.Assert(tree.ExpressionString(sx), Equals, "true")

	sx = Simplify(tree.Comparison{Left: tree.NumericLiteral{42}, Op: tree.LT, Right: tree.NumericLiteral{41}})
	c.Assert(tree.ExpressionString(sx), Equals, "false")

	sx = Simplify(tree.Comparison{Left: tree.NumericLiteral{42}, Op: tree.LTE, Right: tree.NumericLiteral{41}})
	c.Assert(tree.ExpressionString(sx), Equals, "false")
}

func (s *SimplifierSuite) Test_simplifyOr(c *C) {
	sx := Simplify(tree.Or{
		tree.Comparison{
			Left:  tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		},
		tree.Comparison{
			Left:  tree.Variable{"argxxx"},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		},
	})
	c.Assert(tree.ExpressionString(sx), Equals, "(eq argxxx 10)")

	sx = Simplify(tree.Or{
		tree.Comparison{
			Left:  tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.Variable{"foo"}},
		},
		tree.Comparison{
			Left:  tree.Variable{"argxxx"},
			Op:    tree.EQL,
			Right: tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		},
	})
	c.Assert(tree.ExpressionString(sx), Equals, "(or (eq 168 (rsh 42 foo)) (eq argxxx 10))")
}

func (s *SimplifierSuite) Test_Argument(c *C) {
	sx := Simplify(tree.Argument{Index: 3})
	c.Assert(tree.ExpressionString(sx), Equals, "arg3")
}

func (s *SimplifierSuite) Test_simplifyBinaryNegation(c *C) {
	sx := Simplify(tree.BinaryNegation{tree.NumericLiteral{42}})
	c.Assert(tree.ExpressionString(sx), Equals, "18446744073709551573")
}

func (s *SimplifierSuite) Test_simplifyBooleanLiteral(c *C) {
	sx := Simplify(tree.BooleanLiteral{true})
	c.Assert(tree.ExpressionString(sx), Equals, "true")
}

var inclusionSimplifiers = []tree.Transformer{
	createArithmeticSimplifier(),
	createInclusionSimplifier(),
}

func (s *SimplifierSuite) Test_simplifyInclusion(c *C) {
	sx := reduceTransformers(tree.Inclusion{
		Positive: true,
		Left:     tree.BinaryNegation{tree.NumericLiteral{42}},
		Rights: []tree.Numeric{
			tree.NumericLiteral{42},
			tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		}}, inclusionSimplifiers...)
	c.Assert(tree.ExpressionString(sx), Equals, "false")

	sx = reduceTransformers(tree.Inclusion{
		Positive: true,
		Left:     tree.BinaryNegation{tree.NumericLiteral{42}},
		Rights: []tree.Numeric{
			tree.Argument{Index: 0},
			tree.Argument{Index: 2},
			tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		}}, inclusionSimplifiers...)
	c.Assert(tree.ExpressionString(sx), Equals, "(in 18446744073709551573 arg0 arg2)")

	sx = reduceTransformers(tree.Inclusion{
		Positive: true,
		Left:     tree.Argument{Index: 0},
		Rights: []tree.Numeric{
			tree.NumericLiteral{42},
			tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		}}, inclusionSimplifiers...)
	c.Assert(tree.ExpressionString(sx), Equals, "(in arg0 42 168)")
}

func (s *SimplifierSuite) Test_simplifyNotInclusionWithNumericLiteral(c *C) {

	sx := createInclusionSimplifier().Transform(tree.Inclusion{
		Positive: false,
		Left:     tree.Argument{Index: 0},
		Rights: []tree.Numeric{
			tree.NumericLiteral{1},
		}})
	c.Assert(tree.ExpressionString(sx), Equals, "(notIn arg0 1)")
}

func (s *SimplifierSuite) Test_simplifyInclusionWithSameArgumentOnLeftAndRight(c *C) {
	// I'm ok with this weirdness
	sx := createInclusionSimplifier().Transform(tree.Inclusion{
		Positive: true,
		Left:     tree.Argument{Index: 0},
		Rights: []tree.Numeric{
			tree.NumericLiteral{4},
			tree.NumericLiteral{5},
			tree.Argument{Index: 0},
		}})
	c.Assert(tree.ExpressionString(sx), Equals, "(in arg0 4 5 arg0)")
}

func (s *SimplifierSuite) Test_simplifyInclusionWithArgumentInRights(c *C) {
	sx := createInclusionSimplifier().Transform(tree.Inclusion{
		Positive: true,
		Left:     tree.Argument{Index: 0},
		Rights: []tree.Numeric{
			tree.NumericLiteral{4},
			tree.NumericLiteral{5},
			tree.Argument{Index: 1},
		}})
	c.Assert(tree.ExpressionString(sx), Equals, "(in arg0 4 5 arg1)")
}

func (s *SimplifierSuite) Test_simplifyInclusionWithOnlyOneValueShouldBecomeComparison(c *C) {
	sx := reduceTransformers(tree.Inclusion{
		Positive: true,
		Left:     tree.BinaryNegation{tree.NumericLiteral{42}},
		Rights: []tree.Numeric{
			tree.Argument{Index: 0},
			tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{2}},
		}}, inclusionSimplifiers...)
	c.Assert(tree.ExpressionString(sx), Equals, "(eq 18446744073709551573 arg0)")
}

func (s *SimplifierSuite) Test_simplifyInclusionForAllLiteralRights(c *C) {
	sx := reduceTransformers(tree.Inclusion{
		Positive: true,
		Left:     tree.BinaryNegation{tree.NumericLiteral{5}},
		Rights: []tree.Numeric{
			tree.NumericLiteral{2},
			tree.NumericLiteral{3},
			tree.NumericLiteral{4},
		}}, inclusionSimplifiers...)
	c.Assert(tree.ExpressionString(sx), Equals, "false")
}

func (s *SimplifierSuite) Test_simplifyNegation(c *C) {
	sx := Simplify(tree.Negation{tree.BooleanLiteral{true}})
	c.Assert(tree.ExpressionString(sx), Equals, "false")
}

func (s *SimplifierSuite) Test_simplifyNumericLiteral(c *C) {
	sx := Simplify(tree.NumericLiteral{42})
	c.Assert(tree.ExpressionString(sx), Equals, "42")
}

func (s *SimplifierSuite) Test_Variable(c *C) {
	sx := Simplify(tree.Variable{"foo"})
	c.Assert(tree.ExpressionString(sx), Equals, "foo")
}

func (s *SimplifierSuite) Test_Arguments(c *C) {
	t := tree.Argument{Index: 1}
	sx := Simplify(t)
	c.Assert(sx, Equals, t)
}

func (s *SimplifierSuite) Test_BooleanLiteral(c *C) {
	t := tree.BooleanLiteral{true}
	sx := Simplify(t)
	c.Assert(sx, Equals, t)
}

func (s *SimplifierSuite) Test_NumericLiteral(c *C) {
	t := tree.NumericLiteral{42}
	sx := Simplify(t)
	c.Assert(sx, Equals, t)
}
