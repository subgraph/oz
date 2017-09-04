package compiler

import (
	"syscall"
	"testing"

	"github.com/twtiger/gosecco/asm"
	"github.com/twtiger/gosecco/tree"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type NumericCompilerSuite struct{}

var _ = Suite(&NumericCompilerSuite{})

func (s *NumericCompilerSuite) Test_compilationOfLiteral(c *C) {
	p := tree.NumericLiteral{42}
	ctx := createCompilerContext()
	compileNumeric(ctx, p)

	c.Assert(asm.Dump(ctx.result), Equals, "ld_imm	2A\n")
}

func (s *NumericCompilerSuite) Test_compilationOfArgument(c *C) {
	ctx := createCompilerContext()
	compileNumeric(ctx, tree.Argument{Type: tree.Low, Index: 3})
	c.Assert(asm.Dump(ctx.result), Equals, "ld_abs	28\n")

	ctx = createCompilerContext()
	compileNumeric(ctx, tree.Argument{Type: tree.Hi, Index: 1})
	c.Assert(asm.Dump(ctx.result), Equals, "ld_abs	1C\n")
}

func (s *NumericCompilerSuite) Test_simpleAdditionOfNumbers(c *C) {
	ctx := createCompilerContext()
	compileNumeric(ctx, tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{3}, Right: tree.NumericLiteral{42}})
	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_imm	3\n"+
		"add_k	2A\n",
	)
}

// This tests a nested expression:     (((argH1 + 32) * 3) & 42) ^ (argL1 - 15)
func (s *NumericCompilerSuite) Test_moreComplicatedExpression(c *C) {
	ctx := createCompilerContext()
	compileNumeric(ctx,
		tree.Arithmetic{
			Op: tree.BINXOR,
			Left: tree.Arithmetic{
				Op: tree.BINAND,
				Left: tree.Arithmetic{
					Op:    tree.MULT,
					Right: tree.NumericLiteral{3},
					Left: tree.Arithmetic{
						Op:    tree.PLUS,
						Left:  tree.Argument{Type: tree.Hi, Index: 1},
						Right: tree.NumericLiteral{32},
					},
				},
				Right: tree.NumericLiteral{42},
			},
			Right: tree.Arithmetic{
				Op:    tree.MINUS,
				Left:  tree.Argument{Type: tree.Low, Index: 1},
				Right: tree.NumericLiteral{15},
			},
		},
	)
	c.Assert(asm.Dump(ctx.result), Equals, ""+
		"ld_abs	18\n"+
		"sub_k	F\n"+
		"st	0\n"+
		"ld_abs	1C\n"+
		"add_k	20\n"+
		"mul_k	3\n"+
		"and_k	2A\n"+
		"ldx_mem	0\n"+
		"xor_x\n",
	)
}

func (s *NumericCompilerSuite) Test_thatAnErrorIsSetWhenWeCompileInvalidExpression(c *C) {
	ctx := createCompilerContext()
	ctx.stackTop = syscall.BPF_MEMWORDS
	err := compileNumeric(ctx, tree.Arithmetic{Op: tree.PLUS, Left: tree.NumericLiteral{3}, Right: tree.Argument{Type: tree.Low, Index: 1}})
	c.Assert(err, ErrorMatches, "the expression is too complicated to compile. Please refer to the language documentation")
}

func (s *NumericCompilerSuite) Test_thatAnErrorIsSetWhenWeCompileInvalidTypes(c *C) {
	err := compileNumeric(createCompilerContext(), tree.BinaryNegation{})
	c.Assert(err, ErrorMatches, "a binary negation was found in an expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.And{})
	c.Assert(err, ErrorMatches, "an and was found in a numeric expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.BooleanLiteral{})
	c.Assert(err, ErrorMatches, "a boolean literal was found in a numeric expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.Call{})
	c.Assert(err, ErrorMatches, "a call was found in an expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.Comparison{})
	c.Assert(err, ErrorMatches, "a comparison was found in a numeric expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.Inclusion{})
	c.Assert(err, ErrorMatches, "an in-statement was found in an expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.Negation{})
	c.Assert(err, ErrorMatches, "a boolean negation was found in a numeric expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.Or{})
	c.Assert(err, ErrorMatches, "an or was found in a numeric expression - this is likely a programmer error")

	err = compileNumeric(createCompilerContext(), tree.Variable{})
	c.Assert(err, ErrorMatches, "a variable was found in an expression - this is likely a programmer error")
}

func (s *NumericCompilerSuite) Test_arithmeticShouldPassAlongErrorsOnTheRightSide(c *C) {
	err := compileNumeric(createCompilerContext(), tree.Arithmetic{Right: tree.BooleanLiteral{false}, Left: tree.NumericLiteral{42}, Op: tree.PLUS})
	c.Assert(err, ErrorMatches, "a boolean literal was found in a numeric expression - this is likely a programmer error")
}

func (s *NumericCompilerSuite) Test_arithmeticShouldPassAlongErrorsOnTheLeftSide(c *C) {
	err := compileNumeric(createCompilerContext(), tree.Arithmetic{Left: tree.BooleanLiteral{false}, Right: tree.NumericLiteral{42}, Op: tree.PLUS})
	c.Assert(err, ErrorMatches, "a boolean literal was found in a numeric expression - this is likely a programmer error")
}

func (s *NumericCompilerSuite) Test_arithmeticShouldPassAlongErrorsIfWeGetIncorrectOperator(c *C) {
	err := compileNumeric(createCompilerContext(), tree.Arithmetic{Left: tree.NumericLiteral{1}, Right: tree.NumericLiteral{42}, Op: tree.ArithmeticType(42)})
	c.Assert(err, ErrorMatches, "an invalid arithmetic operator was found - this is likely a programmer error")
}

func (s *NumericCompilerSuite) Test_arithmeticShouldPassAlongStackTooLargeError(c *C) {
	ctx := createCompilerContext()
	ctx.stackTop = syscall.BPF_MEMWORDS
	err := compileNumeric(ctx, tree.Arithmetic{Left: tree.NumericLiteral{1}, Right: tree.Argument{Type: tree.Low, Index: 1}, Op: tree.ArithmeticType(42)})
	c.Assert(err, ErrorMatches, "the expression is too complicated to compile. Please refer to the language documentation")
}

func (s *NumericCompilerSuite) Test_arithmeticShouldPassAlongStackTooSmallError(c *C) {
	ctx := createCompilerContext()
	err := compileNumeric(ctx,
		tree.Arithmetic{
			Left: &stackMesser{func() {
				ctx.stackTop = 0
			}},
			Right: tree.Argument{Type: tree.Hi, Index: 0},
			Op:    tree.PLUS})
	c.Assert(err, ErrorMatches, "popping from empty stack - this is likely a programmer error")
}

type stackMesser struct {
	f func()
}

func (x *stackMesser) Accept(v tree.Visitor) {
	x.f()
}
