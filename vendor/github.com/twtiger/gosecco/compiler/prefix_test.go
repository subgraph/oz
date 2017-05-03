package compiler

import (
	"github.com/twtiger/gosecco/asm"
	. "gopkg.in/check.v1"
)

type PrefixSuite struct{}

var _ = Suite(&PrefixSuite{})

func (s *PrefixSuite) Test_compilesAuditArch(c *C) {
	ctx := createCompilerContext()
	ctx.compileAuditArchCheck("kill")
	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t00\tC000003E\n")
}

func (s *PrefixSuite) Test_compiles32ABICheck(c *C) {
	ctx := createCompilerContext()
	ctx.compileX32ABICheck("trace")
	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_abs\t0\n"+
		"jset_k\t00\t00\t40000000\n")
}
