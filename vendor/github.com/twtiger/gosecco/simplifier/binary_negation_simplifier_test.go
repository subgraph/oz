package simplifier

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type BinaryNegationSimplifierSuite struct{}

var _ = Suite(&BinaryNegationSimplifierSuite{})

func (s *BinaryNegationSimplifierSuite) Test_simplifiesBinaryNegation(c *C) {
	sx := createBinaryNegationSimplifier().Transform(tree.BinaryNegation{tree.NumericLiteral{42}})

	c.Assert(tree.ExpressionString(sx), Equals, "(binxor 42 18446744073709551615)") // This big ugly value is 0xFFFFFFFFFFFFFFFF, the largest uint64 value
}
