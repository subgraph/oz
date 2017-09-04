package parser

import (
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type BindingSuite struct{}

var _ = Suite(&BindingSuite{})

func parseBindingHeadCheck(c *C, s string, r tree.Macro) {
	res, ok := parseBindingHead(s)
	c.Assert(ok, Equals, true)
	c.Check(res, DeepEquals, r)
}

func (s *BindingSuite) Test_parseBindingHead_parsesValidBindingHeads(c *C) {
	parseBindingHeadCheck(c, "test1", tree.Macro{Name: "test1"})
	parseBindingHeadCheck(c, " _IOC ", tree.Macro{Name: "_IOC"})
	parseBindingHeadCheck(c, "\t write  ", tree.Macro{Name: "write"})
	parseBindingHeadCheck(c, "fcntl()", tree.Macro{Name: "fcntl"})
	parseBindingHeadCheck(c, "fcntl ( ) ", tree.Macro{Name: "fcntl"})
	parseBindingHeadCheck(c, " fcntl ( kill ) ", tree.Macro{Name: "fcntl", ArgumentNames: []string{"kill"}})
	parseBindingHeadCheck(c, " fcntl( kill, trace) ", tree.Macro{Name: "fcntl", ArgumentNames: []string{"kill", "trace"}})

	_, ok := parseBindingHead("")
	c.Assert(ok, Equals, false)

	_, ok = parseBindingHead("fcntl[hm]")
	c.Assert(ok, Equals, false)
}

func (s *BindingSuite) Test_parseBinding_parseABinding(c *C) {
	res, error := parseBinding("test1=42")
	c.Assert(error, IsNil)
	c.Assert(res.Name, Equals, "test1")
	c.Assert(tree.ExpressionString(res.Body), Equals, "42")
}

func (s *BindingSuite) Test_parseBinding_parseAComplicatedBinding(c *C) {
	res, error := parseBinding("test2(a, b, c)=a + b*c")
	c.Assert(error, IsNil)
	c.Assert(res.Name, Equals, "test2")
	c.Assert(tree.ExpressionString(res.Body), Equals, "(plus a (mul b c))")
}

func (s *BindingSuite) Test_parseBinding_parseFailingBinding(c *C) {
	_, error := parseBinding("test2(a, b, c)=a b")
	c.Assert(error.Error(), Equals, "expression is invalid. unable to parse: expected EOF, found 'IDENT' b")
}

func (s *BindingSuite) Test_parseBinding_parseFailingBindingHead(c *C) {
	_, error := parseBinding("test2[]=a")
	c.Assert(error.Error(), Equals, "Invalid macro name")
}
