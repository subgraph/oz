package parser

import (
	"os"
	"path"
	"strings"

	"github.com/twtiger/gosecco/tree"

	. "gopkg.in/check.v1"
)

type FileSuite struct{}

var _ = Suite(&FileSuite{})

func getActualTestFolder() string {
	wd, _ := os.Getwd()
	if strings.HasSuffix(wd, "/parser") {
		return wd + "/test_policies"
	}
	return path.Join(wd, "parser/test_policies/")
}

func (s *FileSuite) Test_ParseFile(c *C) {
	rp, _ := ParseFile(getActualTestFolder() + "/simple_test_policy")
	c.Assert(rp, DeepEquals, tree.RawPolicy{
		RuleOrMacros: []interface{}{
			tree.Macro{
				Name:          "DEFAULT_POSITIVE",
				ArgumentNames: nil,
				Body:          tree.Variable{Name: "kill"}},
			tree.Macro{
				Name:          "something",
				ArgumentNames: []string{"a"},
				Body:          tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x1}, Right: tree.Variable{Name: "a"}}},
			tree.Macro{
				Name:          "VAL",
				ArgumentNames: nil,
				Body:          tree.NumericLiteral{Value: 0x2a}},
			tree.Rule{
				Name:           "read",
				PositiveAction: "",
				NegativeAction: "",
				Body:           tree.NumericLiteral{Value: 0x2a}},
		}})
}

func (s *FileSuite) Test_ParseString(c *C) {
	example := "# a comment\n" +
		"\n" +
		"DEFAULT_POSITIVE=kill\n" +
		"\n" +
		"something(a) = 1 + a\n" +
		"VAL  = 42\n" +
		"\n" +
		"read: 42\n"

	rp, _ := ParseString(example)
	c.Assert(rp, DeepEquals, tree.RawPolicy{
		RuleOrMacros: []interface{}{
			tree.Macro{
				Name:          "DEFAULT_POSITIVE",
				ArgumentNames: nil,
				Body:          tree.Variable{Name: "kill"}},
			tree.Macro{
				Name:          "something",
				ArgumentNames: []string{"a"},
				Body:          tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x1}, Right: tree.Variable{Name: "a"}}},
			tree.Macro{
				Name:          "VAL",
				ArgumentNames: nil,
				Body:          tree.NumericLiteral{Value: 0x2a}},
			tree.Rule{
				Name:           "read",
				PositiveAction: "",
				NegativeAction: "",
				Body:           tree.NumericLiteral{Value: 0x2a}},
		}})
}

func (s *FileSuite) Test_Parse_fromCombinedSource(c *C) {
	source1 := &FileSource{getActualTestFolder() + "/simple_test_policy"}
	source2 := &StringSource{"<tmp1>", "write: 43"}

	rp, _ := Parse(CombineSources(source1, source2))
	c.Assert(rp, DeepEquals, tree.RawPolicy{
		RuleOrMacros: []interface{}{
			tree.Macro{
				Name:          "DEFAULT_POSITIVE",
				ArgumentNames: nil,
				Body:          tree.Variable{Name: "kill"}},
			tree.Macro{
				Name:          "something",
				ArgumentNames: []string{"a"},
				Body:          tree.Arithmetic{Op: 0, Left: tree.NumericLiteral{Value: 0x1}, Right: tree.Variable{Name: "a"}}},
			tree.Macro{
				Name:          "VAL",
				ArgumentNames: nil,
				Body:          tree.NumericLiteral{Value: 0x2a}},
			tree.Rule{
				Name:           "read",
				PositiveAction: "",
				NegativeAction: "",
				Body:           tree.NumericLiteral{Value: 0x2a}},
			tree.Rule{
				Name:           "write",
				PositiveAction: "",
				NegativeAction: "",
				Body:           tree.NumericLiteral{Value: 0x2b}},
		}})
}

func (s *FileSuite) Test_ParseFile_failing(c *C) {
	rp, ee := ParseFile(getActualTestFolder() + "/failing_test_policy")
	c.Assert(rp.RuleOrMacros, IsNil)
	c.Assert(ee, ErrorMatches, ".*parser/test_policies/failing_test_policy:1: unexpected end of line")
}
