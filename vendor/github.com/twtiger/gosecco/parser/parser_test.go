package parser

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type ParserSuite struct{}

var _ = Suite(&ParserSuite{})

func (s *ParserSuite) Test_parsesNumber(c *C) {
	result, _, _, _ := parseExpression("42")

	c.Assert(result, DeepEquals, tree.NumericLiteral{42})
}

func (s *ParserSuite) Test_parsesAddition(c *C) {
	result, _, _, _ := parseExpression("42 + 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesMultiplication(c *C) {
	result, _, _, _ := parseExpression("42 * 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesDivision(c *C) {
	result, _, _, _ := parseExpression("42 / 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.DIV, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesModulo(c *C) {
	result, _, _, _ := parseExpression("42 % 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.MOD, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesRSH(c *C) {
	result, _, _, _ := parseExpression("42 >> 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.RSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesLSH(c *C) {
	result, _, _, _ := parseExpression("42 << 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.LSH, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesOR(c *C) {
	result, _, _, _ := parseExpression("42 | 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.BINOR, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesAND(c *C) {
	result, _, _, _ := parseExpression("42 & 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.BINAND, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesXOR(c *C) {
	result, _, _, _ := parseExpression("42 ^ 15")

	c.Assert(result, DeepEquals, tree.Arithmetic{Op: tree.BINXOR, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}})
}

func (s *ParserSuite) Test_parsesMultiplicationAndAddition(c *C) {
	result, _, _, _ := parseExpression("42 * 15 + 1")

	c.Assert(result, DeepEquals,
		tree.Arithmetic{Op: tree.PLUS, Left: tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{15}}, Right: tree.NumericLiteral{1}})
}

func (s *ParserSuite) Test_parsesParens(c *C) {
	result, _, _, _ := parseExpression("42 * (15 + 1)")

	c.Assert(result, DeepEquals,
		tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{42}, Right: tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{15}, Right: tree.NumericLiteral{1}}})
}

func (s *ParserSuite) Test_parsesArgument(c *C) {
	result, _, _, _ := parseExpression("42 * arg1")

	c.Assert(result, DeepEquals,
		tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{42}, Right: tree.Argument{Index: 1}})
}

func (s *ParserSuite) Test_parsesVariable(c *C) {
	result, _, _, _ := parseExpression("42 * arg6")

	c.Assert(result, DeepEquals,
		tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{42}, Right: tree.Variable{"arg6"}})
}

func (s *ParserSuite) Test_parsesUnaryNegation(c *C) {
	result, _, _, _ := parseExpression("42 * ~arg6")

	c.Assert(result, DeepEquals,
		tree.Arithmetic{Op: tree.MULT, Left: tree.NumericLiteral{42}, Right: tree.BinaryNegation{tree.Variable{"arg6"}}})
}

func (s *ParserSuite) Test_parsesBooleanExpression(c *C) {
	result, _, _, _ := parseExpression("true && false")
	c.Assert(result, DeepEquals, tree.And{Left: tree.BooleanLiteral{true}, Right: tree.BooleanLiteral{false}})

	result, _, _, _ = parseExpression("true || false")
	c.Assert(result, DeepEquals, tree.Or{Left: tree.BooleanLiteral{true}, Right: tree.BooleanLiteral{false}})

	result, _, _, _ = parseExpression("!(true || false)")
	c.Assert(result, DeepEquals, tree.Negation{tree.Or{Left: tree.BooleanLiteral{true}, Right: tree.BooleanLiteral{false}}})
}

func (s *ParserSuite) Test_parsesEquality(c *C) {
	result, _, _, _ := parseExpression("true == false")
	c.Assert(result, DeepEquals, tree.Comparison{Op: tree.EQL, Left: tree.BooleanLiteral{true}, Right: tree.BooleanLiteral{false}})

	result, _, _, _ = parseExpression("42 != 1")
	c.Assert(result, DeepEquals, tree.Comparison{Op: tree.NEQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}})

	result, _, _, _ = parseExpression("42 > 1")
	c.Assert(result, DeepEquals, tree.Comparison{Op: tree.GT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}})

	result, _, _, _ = parseExpression("42 >= 1")
	c.Assert(result, DeepEquals, tree.Comparison{Op: tree.GTE, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}})

	result, _, _, _ = parseExpression("42 < 1")
	c.Assert(result, DeepEquals, tree.Comparison{Op: tree.LT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}})

	result, _, _, _ = parseExpression("42 <= 1")
	c.Assert(result, DeepEquals, tree.Comparison{Op: tree.LTE, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}})
}

func (s *ParserSuite) Test_parseCall(c *C) {
	result, _, _, _ := parseExpression("foo(1+1, 2+3, arg0)")
	c.Assert(result, DeepEquals,
		tree.Call{Name: "foo",
			Args: []tree.Any{
				tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x1}, Right: tree.NumericLiteral{Value: 0x1}},
				tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x2}, Right: tree.NumericLiteral{Value: 0x3}},
				tree.Argument{Index: 0}}})
}

func (s *ParserSuite) Test_parseIn(c *C) {
	result, _, _, _ := parseExpression("in(1+1, 2+3, arg0)")
	c.Assert(result, DeepEquals,
		tree.Inclusion{Positive: true,
			Left: tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x1}, Right: tree.NumericLiteral{Value: 0x1}},
			Rights: []tree.Numeric{
				tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x2}, Right: tree.NumericLiteral{Value: 0x3}},
				tree.Argument{Index: 0}}})
}

func (s *ParserSuite) Test_parseNotIn(c *C) {
	result, _, _, _ := parseExpression("notin(1+1, 2+3, arg0)")
	c.Assert(result, DeepEquals,
		tree.Inclusion{Positive: false,
			Left: tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x1}, Right: tree.NumericLiteral{Value: 0x1}},
			Rights: []tree.Numeric{
				tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x2}, Right: tree.NumericLiteral{Value: 0x3}},
				tree.Argument{Index: 0}}})
}

func (s *ParserSuite) Test_parsesSimpleRule(c *C) {
	result, _, _, _ := parseExpression("1")

	c.Assert(result, DeepEquals, tree.BooleanLiteral{true})
}

func (s *ParserSuite) Test_parsesAlmostSimpleRule(c *C) {
	result, _, _, _ := parseExpression("arg0 > 0")

	c.Assert(result, DeepEquals, tree.Comparison{
		Left:  tree.Argument{Index: 0},
		Op:    tree.GT,
		Right: tree.NumericLiteral{0},
	})
}

func (s *ParserSuite) Test_parseAnotherRule(c *C) {
	result, _, _, _ := parseExpression("arg0 == 4")

	c.Assert(result, DeepEquals, tree.Comparison{
		Left:  tree.Argument{Index: 0},
		Op:    tree.EQL,
		Right: tree.NumericLiteral{4},
	})
}

func (s *ParserSuite) Test_parseYetAnotherRule(c *C) {
	result, _, _, _ := parseExpression("arg0 == 4 || arg0 == 5")

	c.Assert(tree.ExpressionString(result), Equals, "(or (eq arg0 4) (eq arg0 5))")
	c.Assert(result, DeepEquals, tree.Or{
		Left: tree.Comparison{
			Left:  tree.Argument{Index: 0},
			Op:    tree.EQL,
			Right: tree.NumericLiteral{4},
		},
		Right: tree.Comparison{
			Left:  tree.Argument{Index: 0},
			Op:    tree.EQL,
			Right: tree.NumericLiteral{5},
		},
	})
}

func (s *ParserSuite) Test_parseYetAnotherRuleWithBitsetComparison(c *C) {
	result, _, _, _ := parseExpression("arg0 == 4 || arg0 &? 5")

	c.Assert(tree.ExpressionString(result), Equals, "(or (eq arg0 4) (bitset arg0 5))")
	c.Assert(result, DeepEquals, tree.Or{
		Left: tree.Comparison{
			Left:  tree.Argument{Index: 0},
			Op:    tree.EQL,
			Right: tree.NumericLiteral{4},
		},
		Right: tree.Comparison{
			Left:  tree.Argument{Index: 0},
			Op:    tree.BITSET,
			Right: tree.NumericLiteral{5},
		},
	})
}

func parseExpectSuccess(c *C, str string) string {
	result, _, _, err := parseExpression(str)
	c.Assert(err, IsNil)
	return tree.ExpressionString(result)
}

func (s *ParserSuite) Test_parseExpressionWithMultiplication(c *C) {
	c.Assert(parseExpectSuccess(c, "arg0 == 12 * 3"), Equals, "(eq arg0 (mul 12 3))")

}

func (s *ParserSuite) Test_parseAExpressionWithAddition(c *C) {
	result, _, _, _ := parseExpression("arg0 == 12 + 3")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (plus 12 3))")
}

func (s *ParserSuite) Test_parseAExpressionWithDivision(c *C) {
	result, _, _, _ := parseExpression("arg0 == 12 / 3")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (div 12 3))")
}

func (s *ParserSuite) Test_parseAExpressionWithSubtraction(c *C) {
	result, _, _, _ := parseExpression("arg0 == 12 - 3")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (minus 12 3))")
}

func (s *ParserSuite) Test_parseAExpressionBinaryAnd(c *C) {
	result, _, _, _ := parseExpression("arg0 == 0 & 1")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (binand 0 1))")
}

func (s *ParserSuite) Test_parseAExpressionBinaryOr(c *C) {
	result, _, _, _ := parseExpression("arg0 == 0 | 1")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (binor 0 1))")
}

func (s *ParserSuite) Test_parseAExpressionBinaryXor(c *C) {
	result, _, _, _ := parseExpression("arg0 == 0 ^ 1")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (binxor 0 1))")
}

func (s *ParserSuite) Test_parseAExpressionBinaryNegation(c *C) {
	c.Assert(parseExpectSuccess(c, "arg0 == ~0"), Equals, "(eq arg0 (binNeg 0))")
}

func (s *ParserSuite) Test_parseAExpressionWithBinaryNegationTwice(c *C) {
	c.Assert(parseExpectSuccess(c, "arg0 == ~~0"), Equals, "(eq arg0 (binNeg (binNeg 0)))")
}

func (s *ParserSuite) Test_parseAExpressionLeftShift(c *C) {
	result, _, _, _ := parseExpression("arg0 == 2 << 1")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (lsh 2 1))")
}

func (s *ParserSuite) Test_parseAExpressionRightShift(c *C) {
	result, _, _, _ := parseExpression("arg0 == (2 >> 1)")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (rsh 2 1))")
}

func (s *ParserSuite) Test_parseAExpressionWithModulo(c *C) {
	result, _, _, _ := parseExpression("arg0 == 12 % 3")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (mod 12 3))")
}

func (s *ParserSuite) Test_parseAWeirdThing(c *C) {
	result, _, _, _ := parseExpression("arg0 == a | b")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (binor a b))")
}

func (s *ParserSuite) Test_parseAExpressionWithBooleanAnd(c *C) {
	result, _, _, _ := parseExpression("arg0 == 0 && arg1 == 0")
	c.Assert(tree.ExpressionString(result), Equals, "(and (eq arg0 0) (eq arg1 0))")
}

func (s *ParserSuite) Test_parseAExpressionWithBooleanNegation(c *C) {
	c.Assert(parseExpectSuccess(c, "!(arg0 == 1)"), Equals, "(not (eq arg0 1))")
}

func (s *ParserSuite) Test_parseAExpressionWithDoubleBooleanNegation(c *C) {
	c.Assert(parseExpectSuccess(c, "!!(arg0 == 1)"), Equals, "(not (not (eq arg0 1)))")
}

func (s *ParserSuite) Test_parseAExpressionWithNotEqual(c *C) {
	result, _, _, _ := parseExpression("arg0 != 1")
	c.Assert(tree.ExpressionString(result), Equals, "(neq arg0 1)")
}

func (s *ParserSuite) Test_parseAExpressionWithGreaterThanOrEqualTo(c *C) {
	result, _, _, _ := parseExpression("arg0 >= 1")
	c.Assert(tree.ExpressionString(result), Equals, "(gte arg0 1)")
}

func (s *ParserSuite) Test_parseAExpressionWithLessThan(c *C) {
	result, _, _, _ := parseExpression("arg0 < arg1")
	c.Assert(tree.ExpressionString(result), Equals, "(lt arg0 arg1)")
}

func (s *ParserSuite) Test_parseAExpressionWithLessThanOrEqualTo(c *C) {
	result, _, _, _ := parseExpression("arg0 <= arg1")
	c.Assert(tree.ExpressionString(result), Equals, "(lte arg0 arg1)")
}

func (s *ParserSuite) Test_parseAExpressionWithInclusion(c *C) {
	result, _, _, _ := parseExpression("in(arg0, 1, 2)")
	c.Assert(tree.ExpressionString(result), Equals, "(in arg0 1 2)")
}

func (s *ParserSuite) Test_parseAExpressionWithExclusion(c *C) {
	result, _, _, _ := parseExpression("notIn(arg0, 1, 2)")
	c.Assert(tree.ExpressionString(result), Equals, "(notIn arg0 1 2)")
}

func (s *ParserSuite) Test_parseAExpressionWithInclusionLargerSet(c *C) {
	result, _, _, _ := parseExpression("in(arg0, 1, 2, 3, 4)")
	c.Assert(tree.ExpressionString(result), Equals, "(in arg0 1 2 3 4)")
}

func (s *ParserSuite) Test_parseAExpressionWithAnotherSet(c *C) {
	result, _, _, _ := parseExpression("in(3, 1, 2, 3, 4)")
	c.Assert(tree.ExpressionString(result), Equals, "(in 3 1 2 3 4)")
}

func (s *ParserSuite) Test_parseAExpressionWithExclusionLargerSet(c *C) {
	result, _, _, _ := parseExpression("notIn(arg0, 1, 2, 3, 4)")
	c.Assert(tree.ExpressionString(result), Equals, "(notIn arg0 1 2 3 4)")
}

func (s *ParserSuite) Test_parseAExpressionWithInclusionWithWhitespace(c *C) {
	result, _, _, _ := parseExpression("in(arg0, 1,   2,   3,   4)")
	c.Assert(tree.ExpressionString(result), Equals, "(in arg0 1 2 3 4)")
}

func (s *ParserSuite) Test_parseAExpressionWithTrue(c *C) {
	result, _, _, _ := parseExpression("true")
	c.Assert(tree.ExpressionString(result), Equals, "true")
}

func (s *ParserSuite) Test_parseAExpressionWithFalse(c *C) {
	result, _, _, _ := parseExpression("false")
	c.Assert(tree.ExpressionString(result), Equals, "false")
}

func (s *ParserSuite) Test_parseAExpressionWith1AsTrue(c *C) {
	result, _, _, _ := parseExpression("1")
	c.Assert(tree.ExpressionString(result), Equals, "true")
}

func (s *ParserSuite) Test_parseAExpressionWithParens(c *C) {
	result, _, _, _ := parseExpression("arg0 == (12 + 3) * 2")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (mul (plus 12 3) 2))")
}

func (s *ParserSuite) Test_parseAExpressionWithNestedOperators(c *C) {
	result, _, _, _ := parseExpression("arg0 == 12 + 3 * 2")
	c.Assert(tree.ExpressionString(result), Equals, "(eq arg0 (plus 12 (mul 3 2)))")
}

func (s *ParserSuite) Test_parseAExpressionWithInvalidArithmeticOperator(c *C) {
	_, _, _, err := parseExpression("arg0 == 12 _ 3")
	c.Assert(err, ErrorMatches, "expression is invalid. unable to parse: expected EOF, found 'IDENT' _")
}

func (s *ParserSuite) Test_parseArgumentsCorrectly_andIncorrectly(c *C) {
	c.Assert(parseExpectSuccess(c, "arg0 == 0"), Equals, "(eq arg0 0)")
	c.Assert(parseExpectSuccess(c, "arg5 == 0"), Equals, "(eq arg5 0)")

	result, _, _, _ := parseExpression("arg6 == 0")
	c.Assert(result, DeepEquals, tree.Comparison{
		Left:  tree.Variable{"arg6"},
		Op:    tree.EQL,
		Right: tree.NumericLiteral{0},
	})
}

func (s *ParserSuite) Test_parseBitSet(c *C) {
	c.Assert(parseExpectSuccess(c, "arg0 &? 1"), Equals, "(bitset arg0 1)")
}

func (s *ParserSuite) Test_parseCallInBooleanContext(c *C) {
	c.Assert(parseExpectSuccess(c, "foo(42 - 15, 1+2)"), Equals, "(foo (minus 42 15) (plus 1 2))")
}

func (s *ParserSuite) Test_parseCallInNumericContext(c *C) {
	c.Assert(parseExpectSuccess(c, "15 == foo(42 * 15, 1+2)"), Equals, "(eq 15 (foo (mul 42 15) (plus 1 2)))")
}

func (s *ParserSuite) Test_parseHexNumbers(c *C) {
	c.Assert(parseExpectSuccess(c, "0x12 == 42"), Equals, "(eq 18 42)")
}

func (s *ParserSuite) Test_parseOctalNumbers(c *C) {
	c.Assert(parseExpectSuccess(c, "012 == 42"), Equals, "(eq 10 42)")
}

func (s *ParserSuite) Test_parseSimpleReturn(c *C) {
	x, hasReturn, ret, _ := parseExpression("return 42")

	c.Assert(x, IsNil)
	c.Assert(hasReturn, Equals, true)
	c.Assert(ret, Equals, uint16(42))
}

func (s *ParserSuite) Test_parseComplexReturn(c *C) {
	x, hasReturn, ret, _ := parseExpression("42 == arg0; return 42")
	c.Assert(tree.ExpressionString(x), Equals, "(eq 42 arg0)")
	c.Assert(hasReturn, Equals, true)
	c.Assert(ret, Equals, uint16(42))
}

func (s *ParserSuite) Test_invalidLiteral(c *C) {
	_, _, _, err := parseExpression("arg0 == \"foo\"")
	c.Assert(err, ErrorMatches, "unexpected token at <input>:-1:8: '\"'")
}

func (s *ParserSuite) Test_parsesArgumentPieces(c *C) {
	x, _, _, _ := parseExpression("arg3")
	c.Assert(x, Equals, tree.Argument{Index: 3, Type: tree.Full})

	x, _, _, _ = parseExpression("argH4")
	c.Assert(x, Equals, tree.Argument{Index: 4, Type: tree.Hi})

	x, _, _, _ = parseExpression("argL1")
	c.Assert(x, Equals, tree.Argument{Index: 1, Type: tree.Low})
}

func (s *ParserSuite) Test_invalidCall(c *C) {
	_, _, _, err := parseExpression("foo(1,2,)")
	c.Assert(err, ErrorMatches, "expression is invalid\\. unable to parse: expected primary expression, found '\\)'")
}

func (s *ParserSuite) Test_invalidCall2(c *C) {
	_, _, _, err := parseExpression("foo(1,2,3")
	c.Assert(err, ErrorMatches, "expression is invalid\\. unable to parse: expected '\\)' or ',', found EOF")
}

func (s *ParserSuite) Test_invalidIn(c *C) {
	_, _, _, err := parseExpression("in(1,2,)")
	c.Assert(err, ErrorMatches, "expression is invalid\\. unable to parse: expected primary expression, found '\\)'")
}

func (s *ParserSuite) Test_invalidIn2(c *C) {
	_, _, _, err := parseExpression("in(1,2,3")
	c.Assert(err, ErrorMatches, "expression is invalid\\. unable to parse: expected '\\)' or ',', found EOF")
}

func (s *ParserSuite) Test_invalidIn3(c *C) {
	_, _, _, err := parseExpression("in 2")
	c.Assert(err, ErrorMatches, "expression is invalid\\. unable to parse: expected '\\(', found 'INT' 2")
}

func (s *ParserSuite) Test_invalidParen(c *C) {
	_, _, _, err := parseExpression("(1")
	c.Assert(err, ErrorMatches, "expression is invalid\\. unable to parse: expected '\\)', found EOF")
}

func (s *ParserSuite) Test_emptyString(c *C) {
	x, _, _, err := parseExpression("")
	c.Assert(x, IsNil)
	c.Assert(err, IsNil)
}
