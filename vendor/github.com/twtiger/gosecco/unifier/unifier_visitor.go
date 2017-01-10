package unifier

import (
	"fmt"

	"github.com/twtiger/gosecco/constants"
	"github.com/twtiger/gosecco/tree"
)

type replacer struct {
	expression tree.Expression
	macros     map[string]tree.Macro
	err        error
}

func (r *replacer) AcceptAnd(b tree.And) {
	var left tree.Boolean
	var right tree.Boolean
	left, r.err = replace(b.Left, r.macros)
	if r.err == nil {
		right, r.err = replace(b.Right, r.macros)
		r.expression = tree.And{Left: left, Right: right}
	}
}

func (r *replacer) AcceptArgument(tree.Argument) {}

func (r *replacer) AcceptArithmetic(b tree.Arithmetic) {
	var left tree.Numeric
	var right tree.Numeric
	left, r.err = replace(b.Left, r.macros)
	if r.err == nil {
		right, r.err = replace(b.Right, r.macros)
		r.expression = tree.Arithmetic{Left: left, Op: b.Op, Right: right}
	}
}

func (r *replacer) AcceptBinaryNegation(b tree.BinaryNegation) {
	var op tree.Numeric
	op, r.err = replace(b.Operand, r.macros)
	r.expression = tree.BinaryNegation{op}
}

func (r *replacer) AcceptBooleanLiteral(tree.BooleanLiteral) {}

func (r *replacer) AcceptCall(b tree.Call) {
	v, ok := r.macros[b.Name] // we get the name of the macro

	if !ok {
		r.err = fmt.Errorf("Macro '%s' is not defined", b.Name)
		return
	}

	nm := make(map[string]tree.Macro)
	for i, k := range b.Args {
		var e tree.Expression
		e, r.err = replace(k, r.macros)
		if r.err == nil {
			m := tree.Macro{Name: v.ArgumentNames[i], Body: e}
			nm[v.ArgumentNames[i]] = m
		} else {
			return
		}
	}

	for k, v := range r.macros {
		nm[k] = v
	}

	if r.err == nil {
		r.expression, r.err = replace(v.Body, nm)
	}
}

func (r *replacer) AcceptComparison(b tree.Comparison) {
	var left tree.Numeric
	var right tree.Numeric

	left, r.err = replace(b.Left, r.macros)

	if r.err == nil {
		right, r.err = replace(b.Right, r.macros)
		r.expression = tree.Comparison{
			Left:  left,
			Op:    b.Op,
			Right: right,
		}
	}

}

func (r *replacer) AcceptInclusion(b tree.Inclusion) {
	var rights []tree.Numeric
	for _, e := range b.Rights {
		right, err := replace(e, r.macros)
		if err != nil {
			r.err = err
		}
		rights = append(rights, right)
	}
	left, err := replace(b.Left, r.macros)
	if err != nil {
		r.err = err
	}

	r.expression = tree.Inclusion{Positive: b.Positive, Left: left, Rights: rights}
}

func (r *replacer) AcceptNegation(b tree.Negation) {
	var op tree.Numeric
	op, r.err = replace(b.Operand, r.macros)

	r.expression = tree.Negation{Operand: op}
}

func (r *replacer) AcceptNumericLiteral(tree.NumericLiteral) {}

func (r *replacer) AcceptOr(b tree.Or) {
	var left tree.Boolean
	var right tree.Boolean

	left, r.err = replace(b.Left, r.macros)

	if r.err == nil {
		right, r.err = replace(b.Right, r.macros)
		r.expression = tree.And{Left: left, Right: right}
		r.expression = tree.Or{Left: left, Right: right}
	}
}

func (r *replacer) AcceptVariable(b tree.Variable) {
	expr, ok := r.macros[b.Name]
	if ok {
		x, ee := replace(expr.Body, r.macros)
		if ee != nil {
			r.err = ee
		}
		r.expression = x
	} else {
		value, ok2 := constants.GetConstant(b.Name)
		if ok2 {
			r.expression = tree.NumericLiteral{Value: uint64(value)}
		} else {
			r.err = fmt.Errorf("Variable '%s' is not defined", b.Name)
		}
	}
}
