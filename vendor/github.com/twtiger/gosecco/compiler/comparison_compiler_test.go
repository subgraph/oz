package compiler

import (
	"syscall"

	"github.com/twtiger/gosecco/asm"
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type ComparisonCompilerSuite struct{}

var _ = Suite(&ComparisonCompilerSuite{})

func (s *ComparisonCompilerSuite) Test_SingleComparisons(c *C) {
	ctx := createCompilerContext()

	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
			},
		},
	}

	res, _ := ctx.compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t06\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t02\t1\n"+
		"ld_imm\t2A\n"+
		"jeq_k\t01\t02\t1\n"+
		"jmp\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *ComparisonCompilerSuite) Test_SingleComparisonsArgumentToNumeric(c *C) {
	ctx := createCompilerContext()

	p := tree.Policy{
		DefaultPositiveAction: "allow", DefaultNegativeAction: "kill", DefaultPolicyAction: "kill",
		Rules: []*tree.Rule{
			&tree.Rule{
				Name: "write",
				Body: tree.Comparison{Op: tree.EQL,
					Left:  tree.Argument{Index: 0, Type: tree.Low},
					Right: tree.NumericLiteral{1}},
			},
		},
	}

	res, _ := ctx.compile(p)
	c.Assert(asm.Dump(res), Equals, ""+
		"ld_abs\t4\n"+
		"jeq_k\t00\t06\tC000003E\n"+
		"ld_abs\t0\n"+
		"jeq_k\t00\t02\t1\n"+
		"ld_abs\t10\n"+
		"jeq_k\t01\t02\t1\n"+
		"jmp\t1\n"+
		"ret_k\t7FFF0000\n"+
		"ret_k\t0\n")
}

func (s *ComparisonCompilerSuite) Test_comparisonShouldPassAlongErrorsOnTheRightSide(c *C) {
	p := tree.Comparison{
		Right: tree.BooleanLiteral{false},
		Left:  tree.NumericLiteral{42},
		Op:    tree.GT,
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a boolean literal was found in a numeric expression - this is likely a programmer error")
}

func (s *ComparisonCompilerSuite) Test_comparisonShouldPassAlongErrorsOnTheLeftSide(c *C) {
	p := tree.Comparison{
		Left:  tree.BooleanLiteral{false},
		Right: tree.NumericLiteral{42},
		Op:    tree.GT,
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a boolean literal was found in a numeric expression - this is likely a programmer error")
}

func (s *ComparisonCompilerSuite) Test_comparisonShouldPassAlongStackTooLargeErrors(c *C) {
	p := tree.Comparison{
		Left:  tree.NumericLiteral{1},
		Right: tree.NumericLiteral{42},
		Op:    tree.GT,
	}

	ctx := createCompilerContext()
	ctx.stackTop = syscall.BPF_MEMWORDS
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "the expression is too complicated to compile. Please refer to the language documentation")
}

func (s *ComparisonCompilerSuite) Test_comparisonShouldPassAlongStackTooSmallErrors(c *C) {
	ctx := createCompilerContext()
	p := tree.Comparison{
		Left: &stackMesser{func() {
			ctx.stackTop = 0
		}},
		Right: tree.NumericLiteral{42},
		Op:    tree.GT,
	}

	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "popping from empty stack - this is likely a programmer error")
}

func (s *ComparisonCompilerSuite) Test_comparisonShouldPassAlongErrorsWithIncorrectOperator(c *C) {
	p := tree.Comparison{
		Left:  tree.NumericLiteral{1},
		Right: tree.NumericLiteral{42},
		Op:    tree.LT,
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "this comparison type is not allowed - this is probably a programmer error: \\(lt 1 42\\)")
}
