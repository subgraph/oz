package unifier

import (
	"testing"

	"github.com/twtiger/gosecco/tree"

	. "gopkg.in/check.v1"
)

func ActionsTest(t *testing.T) { TestingT(t) }

type UnifierActionsSuite struct{}

var _ = Suite(&UnifierActionsSuite{})

func (s *UnifierActionsSuite) Test_Unify_setsDefaultActions1(c *C) {
	input := tree.RawPolicy{RuleOrMacros: []interface{}{}}

	output, _ := Unify(input, nil, "allow", "kill", "trace")

	c.Assert(output.DefaultPositiveAction, Equals, "allow")
	c.Assert(output.DefaultNegativeAction, Equals, "kill")
	c.Assert(output.DefaultPolicyAction, Equals, "trace")
}

func (s *UnifierActionsSuite) Test_Unify_setsDefaultActions2(c *C) {
	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{},
	}

	output, _ := Unify(input, nil, "kill", "allow", "trace")

	c.Assert(output.DefaultPositiveAction, Equals, "kill")
	c.Assert(output.DefaultNegativeAction, Equals, "allow")
	c.Assert(output.DefaultPolicyAction, Equals, "trace")
}
