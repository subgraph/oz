package compiler

import (
	"syscall"

	"github.com/twtiger/gosecco/asm"
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type CompilerSuite struct{}

var _ = Suite(&CompilerSuite{})

func (s *CompilerSuite) Test_simplestCompilation(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.BooleanLiteral{true},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs	0\n"+
		"jeq_k	00	01	1\n"+
		"ret_k	7FFF0000\n"+
		"ret_k	0\n")
}

func (s *CompilerSuite) Test_nextSimplestCompilation(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.BooleanLiteral{true},
			},
			&tree.Rule{
				Name: "vhangup",
				Body: tree.BooleanLiteral{true},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t04\tC000003E\n"+
		"ld_abs	0\n"+
		"jeq_k	01	00	1\n"+
		"jeq_k	00	01	99\n"+
		"ret_k	7FFF0000\n"+
		"ret_k	0\n")
}

func (s *CompilerSuite) Test_stackOverflowCreatesError(c *C) {
	cx := createCompilerContext()
	cx.stackTop = syscall.BPF_MEMWORDS
	c.Assert(cx.pushAToStack(), ErrorMatches, "the expression is too complicated to compile. Please refer to the language documentation")
}

func (s *CompilerSuite) Test_stackDoesNotOverflowRightBeforeItsLimit(c *C) {
	cx := createCompilerContext()
	cx.stackTop = syscall.BPF_MEMWORDS - 1
	c.Assert(cx.pushAToStack(), IsNil)
}

func (s *CompilerSuite) Test_stackDoesNotPopAfterReachingTheLowestIndex(c *C) {
	cx := createCompilerContext()
	cx.stackTop = 0
	c.Assert(cx.popStackToX(), ErrorMatches, "popping from empty stack - this is likely a programmer error")
}

func (s *CompilerSuite) Test_compilationOfRuleWithDefinedNegativeAction(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name:           "write",
				NegativeAction: "trace",
				Body:           tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t06\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k	00	02	1\n"+
		"ld_imm	2A\n"+
		"jeq_k\t01\t03\t1\n"+
		"jmp\t1\n"+
		"ret_k	7FFF0000\n"+
		"ret_k\t0\n"+
		"ret_k	7FF00000\n")
}

func (s *CompilerSuite) Test_policyWithDefaultAction(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "allow",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.BooleanLiteral{true},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs	0\n"+
		"jeq_k	00	00	1\n"+
		"ret_k	7FFF0000\n"+
		"ret_k	0\n")
}

func (s *CompilerSuite) Test_policyWithAnotherDefaultAction(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "trace",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.BooleanLiteral{true},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t03\tC000003E\n"+
		"ld_abs	0\n"+
		"jeq_k	00	02	1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k	0\n"+
		"ret_k	7FF00000\n")
}

func (s *CompilerSuite) Test_policyWithDefaultPositiveAction(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "trace", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.BooleanLiteral{true},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t02\tC000003E\n"+
		"ld_abs	0\n"+
		"jeq_k	01	00	1\n"+
		"ret_k	0\n"+
		"ret_k	7FF00000\n")
}

func (s *CompilerSuite) Test_policyWithNegativeDefaultAction(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "trace", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t06\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t02\t1\n"+
		"ld_imm\t2A\n"+
		"jeq_k\t01\t03\t1\n"+
		"jmp\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n"+
		"ret_k\t7FF00000\n")
}
