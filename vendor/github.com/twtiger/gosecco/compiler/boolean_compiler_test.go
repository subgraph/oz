package compiler

import (
	"syscall"

	"github.com/twtiger/gosecco/asm"
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

type BooleanCompilerSuite struct{}

var _ = Suite(&BooleanCompilerSuite{})

func (s *BooleanCompilerSuite) Test_compilationOfSimpleComparison(c *C) {
	p := tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"pos": []int{4},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"neg": []int{4},
	}))
}

func (s *BooleanCompilerSuite) Test_compilationOfSimpleComparison2(c *C) {
	p := tree.Comparison{Op: tree.NEQL, Left: tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{25}}, Right: tree.NumericLiteral{1}}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "posx", "negx")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"add_k	19\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"negx": []int{5},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"posx": []int{5},
	}))
}

func (s *BooleanCompilerSuite) Test_compilationOfSimpleComparison3(c *C) {
	p := tree.Comparison{Op: tree.GT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"ldx_mem	0\n"+
		"jgt_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"pos": []int{4},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"neg": []int{4},
	}))
}

func (s *BooleanCompilerSuite) Test_compilationOfSimpleComparison4(c *C) {
	p := tree.Comparison{Op: tree.GTE, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"ldx_mem	0\n"+
		"jge_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"pos": []int{4},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"neg": []int{4},
	}))
}

func (s *BooleanCompilerSuite) Test_compilationOfInvalidComparison(c *C) {
	p := tree.Comparison{Op: tree.LT, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}}
	ctx := createCompilerContext()
	res := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(res, Not(IsNil))
	c.Assert(res, ErrorMatches, "this comparison type is not allowed - this is probably a programmer error: \\(lt 42 1\\)")
}

func (s *CompilerSuite) Test_topLevelBoolean(c *C) {
	p := tree.BooleanLiteral{true}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, true, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"jmp\t0\n",
	)
}

func (s *BooleanCompilerSuite) Test_compilationOfSimpleAnd(c *C) {
	p := tree.And{
		Left:  tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
		Right: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{41}, Right: tree.NumericLiteral{23}},
	}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n"+
		"ld_imm	17\n"+
		"st	0\n"+
		"ld_imm	29\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"pos":               []int{9},
		"generatedLabel000": []int{4},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"neg": []int{4, 9},
	}))
	c.Assert(ctx.labels, DeepEquals, labelMapFrom(map[label]int{
		"generatedLabel000": 5,
	}))
}

func (s *BooleanCompilerSuite) Test_compilationOfSimpleOr(c *C) {
	p := tree.Or{
		Left:  tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
		Right: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{41}, Right: tree.NumericLiteral{23}},
	}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n"+
		"ld_imm	17\n"+
		"st	0\n"+
		"ld_imm	29\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"pos": []int{4, 9},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"generatedLabel000": []int{4},
		"neg":               []int{9},
	}))
	c.Assert(ctx.labels, DeepEquals, labelMapFrom(map[label]int{
		"generatedLabel000": 5,
	}))
}

func (s *BooleanCompilerSuite) Test_compilationOfSimpleNegation(c *C) {
	p := tree.Negation{
		Operand: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
	}
	ctx := createCompilerContext()
	compileBoolean(ctx, p, false, "pos", "neg")

	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	1\n"+
		"st	0\n"+
		"ld_imm	2A\n"+
		"ldx_mem	0\n"+
		"jeq_x	00	00\n",
	)
	c.Assert(ctx.jts, DeepEquals, jumpMapFrom(map[label][]int{
		"neg": []int{4},
	}))
	c.Assert(ctx.jfs, DeepEquals, jumpMapFrom(map[label][]int{
		"pos": []int{4},
	}))
	c.Assert(ctx.labels, DeepEquals, labelMapFrom(map[label]int{}))
}

func (s *BooleanCompilerSuite) Test_thatAnErrorIsSetWhenWeCompileAfterReachingTheMaximumHeightOfTheStack(c *C) {
	ctx := createCompilerContext()
	ctx.stackTop = syscall.BPF_MEMWORDS
	p := tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}}
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "the expression is too complicated to compile. Please refer to the language documentation")
}

func (s *BooleanCompilerSuite) Test_thatErrorsInLeftHandSidePropagatesfromAnd(c *C) {
	p := tree.And{
		Left:  tree.BooleanLiteral{false},
		Right: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a boolean literal was found in an expression - this is likely a programmer error")
}

func (s *BooleanCompilerSuite) Test_thatErrorsInRightHandSidePropagatesfromAnd(c *C) {
	p := tree.And{
		Right: tree.BooleanLiteral{false},
		Left:  tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a boolean literal was found in an expression - this is likely a programmer error")
}

func (s *BooleanCompilerSuite) Test_thatErrorsInLeftHandSidePropagatesfromOr(c *C) {
	p := tree.Or{
		Left:  tree.BooleanLiteral{false},
		Right: tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a boolean literal was found in an expression - this is likely a programmer error")
}

func (s *BooleanCompilerSuite) Test_thatErrorsInRightHandSidePropagatesfromOr(c *C) {
	p := tree.Or{
		Right: tree.BooleanLiteral{false},
		Left:  tree.Comparison{Op: tree.EQL, Left: tree.NumericLiteral{42}, Right: tree.NumericLiteral{1}},
	}

	ctx := createCompilerContext()
	err := compileBoolean(ctx, p, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a boolean literal was found in an expression - this is likely a programmer error")
}

func (s *BooleanCompilerSuite) Test_thatCompilingInvalidBooleanTypesGeneratesErrors(c *C) {
	err := compileBoolean(createCompilerContext(), tree.Variable{"foo"}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a variable was found in an expression - this is likely a programmer error")

	err = compileBoolean(createCompilerContext(), tree.NumericLiteral{42}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a numeric literal was found in a boolean expression - this is likely a programmer error")

	err = compileBoolean(createCompilerContext(), tree.Argument{Index: 0}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "an argument variable was found in a boolean expression - this is likely a programmer error")

	err = compileBoolean(createCompilerContext(), tree.Call{}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a call was found in an expression - this is likely a programmer error")

	err = compileBoolean(createCompilerContext(), tree.Inclusion{}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "an in-statement was found in an expression - this is likely a programmer error")

	err = compileBoolean(createCompilerContext(), tree.BinaryNegation{}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "a binary negation was found in a boolean expression - this is likely a programmer error")

	err = compileBoolean(createCompilerContext(), tree.Arithmetic{}, false, "pos", "neg")
	c.Assert(err, ErrorMatches, "arithmetic was found in a boolean expression - this is likely a programmer error")
}
