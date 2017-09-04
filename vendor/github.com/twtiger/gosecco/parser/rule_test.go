package parser

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type RuleSuite struct{}

var _ = Suite(&RuleSuite{})

func parseRuleHeadCheck(c *C, s string, r tree.Rule) {
	res, ok := parseRuleHead(s)
	c.Assert(ok, Equals, true)
	c.Check(res, Equals, r)
}

func (s *RuleSuite) Test_parseRuleHead_parsesValidRuleHeads(c *C) {
	parseRuleHeadCheck(c, "read", tree.Rule{Name: "read"})
	parseRuleHeadCheck(c, "write", tree.Rule{Name: "write"})
	parseRuleHeadCheck(c, "\t write  ", tree.Rule{Name: "write"})
	parseRuleHeadCheck(c, "fcntl[]", tree.Rule{Name: "fcntl"})
	parseRuleHeadCheck(c, "fcntl [ ] ", tree.Rule{Name: "fcntl"})
	parseRuleHeadCheck(c, " fcntl [ +kill ] ", tree.Rule{Name: "fcntl", PositiveAction: "kill"})
	parseRuleHeadCheck(c, " fcntl[ -kill] ", tree.Rule{Name: "fcntl", NegativeAction: "kill"})
	parseRuleHeadCheck(c, " fcntl[ -kill, +trace] ", tree.Rule{Name: "fcntl", NegativeAction: "kill", PositiveAction: "trace"})
	parseRuleHeadCheck(c, " fcntl[+trace,-kill] ", tree.Rule{Name: "fcntl", NegativeAction: "kill", PositiveAction: "trace"})
	parseRuleHeadCheck(c, " fcntl[+trace,-42] ", tree.Rule{Name: "fcntl", NegativeAction: "42", PositiveAction: "trace"})

	_, ok := parseRuleHead("")
	c.Assert(ok, Equals, false)

	_, ok = parseRuleHead("fcntl[+trace,+42]")
	c.Assert(ok, Equals, false)

	_, ok = parseRuleHead("fcntl[-trace,-42]")
	c.Assert(ok, Equals, false)

	_, ok = parseRuleHead("fcntl[hm]")
	c.Assert(ok, Equals, false)
}

func (s *RuleSuite) Test_parseRule_returnsErrorForInvalidLine(c *C) {
	_, err := parseRule("  read:  ")
	c.Assert(err, ErrorMatches, "No expression specified for rule: read")
}
