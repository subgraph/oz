package compiler

import (
	"fmt"

	"github.com/twtiger/gosecco/tree"
)

// This is part of the boolean compiler visitor, but needs its own file because of some of the complications involved

var compOps = map[tree.ComparisonType]uint16{
	tree.EQL:  OP_JEQ_X,
	tree.NEQL: OP_JEQ_X,
	tree.GT:   OP_JGT_X,
	tree.GTE:  OP_JGE_X,
}

// AcceptComparison implements Visitor
func (s *booleanCompilerVisitor) AcceptComparison(v tree.Comparison) {
	// At this point in the cycle, only EQL, NEQL, GT and GTE are valid comparisons
	if err := compileNumeric(s.ctx, v.Right); err != nil {
		s.err = err
		return
	}

	if err := s.ctx.pushAToStack(); err != nil {
		s.err = err
		return
	}

	if err := compileNumeric(s.ctx, v.Left); err != nil {
		s.err = err
		return
	}

	if err := s.ctx.popStackToX(); err != nil {
		s.err = err
		return
	}

	compOp, ok := compOps[v.Op]
	if !ok {
		s.err = fmt.Errorf("this comparison type is not allowed - this is probably a programmer error: %s", tree.ExpressionString(v))
		return
	}

	jt, jf := s.jt, s.jf
	if v.Op == tree.NEQL {
		jt, jf = jf, jt
	}

	s.ctx.opWithJumps(compOp, 0, jt, jf)
}
