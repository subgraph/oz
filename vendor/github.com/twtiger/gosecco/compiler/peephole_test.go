package compiler

import (
	"github.com/twtiger/gosecco/asm"
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type PeepholeSuite struct{}

var _ = Suite(&PeepholeSuite{})

func (s *PeepholeSuite) Test_triggeringJumpPeephole(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.BooleanLiteral{true},
			},
			&tree.Rule{
				Name: "read",
				Body: tree.BooleanLiteral{true},
			},
			&tree.Rule{
				Name: "ioctl",
				Body: tree.BooleanLiteral{true},
			},
			&tree.Rule{
				Name: "getrandom",
				Body: tree.BooleanLiteral{true},
			},
		},
	}

	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t06\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t03\t00\t1\n"+
		"jeq_k\t02\t00\t0\n"+
		"jeq_k\t01\t00\t10\n"+
		"jeq_k\t00\t01\t13E\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *PeepholeSuite) Test_triggeringArithmeticStackPeephole(c *C) {
	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.Comparison{
					Op: tree.EQL,
					Left: tree.Arithmetic{
						Op:    tree.PLUS,
						Left:  tree.Argument{Type: tree.Low, Index: 0},
						Right: tree.NumericLiteral{1}},
					Right: tree.NumericLiteral{2},
				},
			},
		},
	}
	res, _ := Compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t0A\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t06\t1\n"+
		"ld_imm\t2\n"+
		"st\t0\n"+
		"ld_abs\t10\n"+
		"add_k\t1\n"+
		"ldx_mem\t0\n"+
		"jeq_x\t01\t02\n"+
		"jmp\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}
