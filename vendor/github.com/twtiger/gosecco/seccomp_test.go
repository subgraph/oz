package gosecco

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/twtiger/gosecco/asm"
	"golang.org/x/sys/unix"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SeccompSuite struct{}

var _ = Suite(&SeccompSuite{})

func (s *SeccompSuite) Test_loadingTooBigBpf(c *C) {
	inp := make([]unix.SockFilter, 0xFFFF+1)
	res := Load(inp)
	c.Assert(res, ErrorMatches, "filter program too big: 65536 bpf instructions \\(limit = 65535\\)")
}

func getActualTestFolder() string {
	wd, _ := os.Getwd()
	if strings.HasSuffix(wd, "/parser/test_policies/") {
		return wd
	}
	return path.Join(wd, "parser/test_policies/")
}

func (s *SeccompSuite) Test_parseInvalidFileReturnsErrors(c *C) {
	set := SeccompSettings{}
	f := getActualTestFolder() + "/failing_test_policy"
	_, ee := Prepare(f, set)
	c.Assert(ee, ErrorMatches, ".*parser/test_policies/failing_test_policy:1: unexpected end of line")
}

func (s *SeccompSuite) Test_parseUnificationErrorReturnsError(c *C) {
	set := SeccompSettings{}
	f := getActualTestFolder() + "/missing_variable_policy"
	_, ee := Prepare(f, set)
	c.Assert(ee, ErrorMatches, "Variable 'b' is not defined")
}

func (s *SeccompSuite) Test_parseValidPolicyFile(c *C) {
	set := SeccompSettings{DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill"}
	f := getActualTestFolder() + "/valid_test_policy"
	res, ee := Prepare(f, set)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t01\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *SeccompSuite) Test_parseInvalidTypeReturnsError(c *C) {
	set := SeccompSettings{}
	f := getActualTestFolder() + "/type_checker_error_policy"
	_, ee := Prepare(f, set)
	c.Assert(ee, ErrorMatches, ".*expected boolean expression but found: 42")
}

func (s *SeccompSuite) Test_parseSimplifiesValidExpression(c *C) {
	set := SeccompSettings{DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill"}
	f := getActualTestFolder() + "/valid_unsimplified_policy"
	res, ee := Prepare(f, set)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t08\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t04\t1\n"+
		"ld_abs\t10\n"+
		"jeq_k\t00\t04\t3\n"+
		"ld_abs\t14\n"+
		"jeq_k\t01\t02\t0\n"+
		"jmp\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *SeccompSuite) Test_parseSetsDefaultActions(c *C) {
	set := SeccompSettings{DefaultPositiveAction: "kill", DefaultNegativeAction: "trace", DefaultPolicyAction: "trace"}
	f := getActualTestFolder() + "/valid_test_policy"
	res, ee := Prepare(f, set)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t02\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t01\t1\n"+
		"ret_k\t0\n"+
		"ret_k\t7FF00000\n")
}

func (s *SeccompSuite) Test_compileWithEnforce(c *C) {
	f := getActualTestFolder() + "/valid_test_policy"
	res, ee := Compile(f, true)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t01\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *SeccompSuite) Test_compileWithoutEnforce(c *C) {
	f := getActualTestFolder() + "/valid_test_policy"
	res, ee := Compile(f, false)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t02\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n"+
		"ret_k\t7FF00000\n")
}

func (s *SeccompSuite) Test_compileBlacklistWithEnforce(c *C) {
	f := getActualTestFolder() + "/valid_test_policy"
	res, ee := CompileBlacklist(f, true)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t01\t00\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *SeccompSuite) Test_compileBlacklistWithoutEnforce(c *C) {
	f := getActualTestFolder() + "/valid_test_policy"
	res, ee := CompileBlacklist(f, false)

	c.Assert(ee, Equals, nil)

	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t02\t00\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n"+
		"ret_k\t7FF00000\n")
}

func (s *SeccompSuite) Test_emptySyscallExpression(c *C) {
	f := getActualTestFolder() + "/empty_expression_test_policy"
	_, ee := Prepare(f, SeccompSettings{})

	c.Assert(ee, ErrorMatches, ".*?No expression specified for rule: write")
}
