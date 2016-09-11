package compiler

import (
	"errors"

	"github.com/twtiger/gosecco/tree"
)

// The boolean compiler uses the stack for simplicity, but we could probably do without
// It generates suboptimal code, expecting a peephole stage after
// It will always take jump points as arguments
// Jump points are arbitary types that represents where to jump.
// All the different boolean situations can be represented using this structure.
// The conditional compiler is also a boolean compiler

type booleanCompilerVisitor struct {
	ctx      *compilerContext
	err      error
	topLevel bool
	jt       label
	jf       label
}

func compileBoolean(ctx *compilerContext, inp tree.Expression, topLevel bool, jt, jf label) error {
	v := &booleanCompilerVisitor{ctx: ctx, jt: jt, jf: jf, topLevel: topLevel}
	inp.Accept(v)
	return v.err
}

func (s *booleanCompilerVisitor) AcceptAnd(v tree.And) {
	next := s.ctx.newLabel()

	if err := compileBoolean(s.ctx, v.Left, false, next, s.jf); err != nil {
		s.err = err
		return
	}

	s.ctx.labelHere(next)

	if err := compileBoolean(s.ctx, v.Right, false, s.jt, s.jf); err != nil {
		s.err = err
		return
	}
}

// AcceptArgument implements Visitor
func (s *booleanCompilerVisitor) AcceptArgument(v tree.Argument) {
	s.err = errors.New("an argument variable was found in a boolean expression - this is likely a programmer error")
}

// AcceptArithmetic implements Visitor
func (s *booleanCompilerVisitor) AcceptArithmetic(v tree.Arithmetic) {
	s.err = errors.New("arithmetic was found in a boolean expression - this is likely a programmer error")
}

// AcceptBinaryNegation implements Visitor
func (s *booleanCompilerVisitor) AcceptBinaryNegation(v tree.BinaryNegation) {
	s.err = errors.New("a binary negation was found in a boolean expression - this is likely a programmer error")
}

// AcceptBooleanLiteral implements Visitor
func (s *booleanCompilerVisitor) AcceptBooleanLiteral(v tree.BooleanLiteral) {
	if s.topLevel {
		s.ctx.unconditionalJumpTo(s.jt)
	} else {
		s.err = errors.New("a boolean literal was found in an expression - this is likely a programmer error")
	}
}

// AcceptCall implements Visitor
func (s *booleanCompilerVisitor) AcceptCall(v tree.Call) {
	s.err = errors.New("a call was found in an expression - this is likely a programmer error")
}

// AcceptInclusion implements Visitor
func (s *booleanCompilerVisitor) AcceptInclusion(v tree.Inclusion) {
	s.err = errors.New("an in-statement was found in an expression - this is likely a programmer error")
}

// AcceptNegation implements Visitor
func (s *booleanCompilerVisitor) AcceptNegation(v tree.Negation) {
	s.err = compileBoolean(s.ctx, v.Operand, false, s.jf, s.jt)
}

// AcceptNumericLiteral implements Visitor
func (s *booleanCompilerVisitor) AcceptNumericLiteral(v tree.NumericLiteral) {
	s.err = errors.New("a numeric literal was found in a boolean expression - this is likely a programmer error")
}

// AcceptOr implements Visitor
func (s *booleanCompilerVisitor) AcceptOr(v tree.Or) {
	next := s.ctx.newLabel()

	if err := compileBoolean(s.ctx, v.Left, false, s.jt, next); err != nil {
		s.err = err
		return
	}

	s.ctx.labelHere(next)
	if err := compileBoolean(s.ctx, v.Right, false, s.jt, s.jf); err != nil {
		s.err = err
		return
	}
}

// AcceptVariable implements Visitor
func (s *booleanCompilerVisitor) AcceptVariable(v tree.Variable) {
	s.err = errors.New("a variable was found in an expression - this is likely a programmer error")
}
