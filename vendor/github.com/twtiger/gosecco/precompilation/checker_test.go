package precompilation

import (
	"testing"

	"github.com/twtiger/gosecco/tree"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type PrecompilationCheckerSuite struct{}

var _ = Suite(&PrecompilationCheckerSuite{})

func (s *PrecompilationCheckerSuite) Test_noFullArguments(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.Argument{Type: tree.Low, Index: 0}, Right: tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 0)

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.Argument{Type: tree.Full, Index: 0}, Right: tree.NumericLiteral{42}}},
	}}

	val = EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no full arguments allowed - this is probably a programmer error: arg0")

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Right: tree.Argument{Type: tree.Full, Index: 2}, Left: tree.NumericLiteral{42}}},
	}}

	val = EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no full arguments allowed - this is probably a programmer error: arg2")
}

func (s *PrecompilationCheckerSuite) Test_noLargeLiterals(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{0xFFFFFFFF}, Right: tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 0)

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{0xFFFFFFFF + 1}, Right: tree.NumericLiteral{42}}},
	}}

	val = EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no literals larger than 0xFFFFFFFF allowed - this is probably a programmer error: 0x100000000")

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Right: tree.NumericLiteral{0xFFFFFFFF + 1}, Left: tree.NumericLiteral{42}}},
	}}

	val = EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no literals larger than 0xFFFFFFFF allowed - this is probably a programmer error: 0x100000000")
}

func (s *PrecompilationCheckerSuite) Test_noVariables(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.Variable{"blah"}, Right: tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no variables allowed - this is probably a programmer error: blah")

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.Variable{"blah2"}}},
	}}

	val = EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no variables allowed - this is probably a programmer error: blah2")
}

func (s *PrecompilationCheckerSuite) Test_noBinaryNegation(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.BinaryNegation{tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no binary negation expressions allowed - this is probably a programmer error: \\(binNeg 42\\)")
}

func (s *PrecompilationCheckerSuite) Test_noCalls(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.Call{Name: "blah"}, Right: tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no calls allowed - this is probably a programmer error: \\(blah\\)")

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.Call{Name: "blah2"}}},
	}}

	val = EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no calls allowed - this is probably a programmer error: \\(blah2\\)")
}

func (s *PrecompilationCheckerSuite) Test_noInStatements(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Inclusion{Positive: true, Left: tree.NumericLiteral{42}, Rights: []tree.Numeric{tree.NumericLiteral{42}}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no inclusion expressions allowed - this is probably a programmer error: \\(in 42 42\\)")
}

func (s *PrecompilationCheckerSuite) Test_noNotInStatements(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Inclusion{Positive: false, Left: tree.NumericLiteral{42}, Rights: []tree.Numeric{tree.NumericLiteral{42}}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no negative inclusion expressions allowed - this is probably a programmer error: \\(notIn 42 42\\)")
}

func (s *PrecompilationCheckerSuite) Test_noLtComparisonsAllowed(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.LT, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no less than comparisons allowed - this is probably a programmer error: \\(lt 1 42\\)")
}

func (s *PrecompilationCheckerSuite) Test_noLteComparisonsAllowed(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.LTE, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}}},
	}}

	val := EnsureValid(toCheck)

	c.Assert(len(val), Equals, 1)
	c.Assert(val[0], ErrorMatches, "\\[read\\] no less than or equals comparisons allowed - this is probably a programmer error: \\(lte 1 42\\)")
}

func (s *PrecompilationCheckerSuite) Test_otherComparisonsAllowed(c *C) {
	toCheck := tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}}},
	}}

	c.Assert(len(EnsureValid(toCheck)), Equals, 0)

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.NEQL, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}}},
	}}

	c.Assert(len(EnsureValid(toCheck)), Equals, 0)

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.GT, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}}},
	}}

	c.Assert(len(EnsureValid(toCheck)), Equals, 0)

	toCheck = tree.Policy{Rules: []*tree.Rule{
		&tree.Rule{Name: "read", Body: tree.Comparison{Op: tree.GTE, Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}}},
	}}

	c.Assert(len(EnsureValid(toCheck)), Equals, 0)
}
