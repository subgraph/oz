package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/twtiger/gosecco/tree"
)

var (
	allowRE      = regexp.MustCompile(`^ *1$`)
	returnRE     = regexp.MustCompile(`^ *return *([[:word:]]+)$`)
	exprReturnRE = regexp.MustCompile(`; *return *([[:word:]]+)$`)
)

type traceData struct {
	file string
	line int
}

type parser struct {
	forBinding bool
	traceData  *traceData
}

func parseExpressionForBinding(expr string) (tree.Expression, bool, uint16, error) {
	return newParser(true, nil).parseExpression(expr)
}

func parseExpression(expr string) (tree.Expression, bool, uint16, error) {
	return newParser(false, nil).parseExpression(expr)
}

func newParser(forBinding bool, td *traceData) *parser {
	return &parser{
		forBinding,
		td,
	}
}

func (p *parser) parseSpecialCases(expr string) (tree.Expression, bool, uint16, bool, string, error) {
	hasRet := false
	ret := uint16(0)
	newExpr := expr
	if !p.forBinding {
		if match := allowRE.FindStringSubmatch(expr); match != nil {
			return tree.BooleanLiteral{true}, false, 0, true, newExpr, nil
		}

		if match := returnRE.FindStringSubmatch(expr); match != nil {
			errno, err := strconv.ParseUint(match[1], 0, 16)
			if err == nil {
				return nil, true, uint16(errno), true, newExpr, nil
			}
			return nil, false, 0, true, newExpr, err
		}

		if match := exprReturnRE.FindStringSubmatch(expr); match != nil {
			newExpr = strings.TrimSuffix(expr, match[0])
			errno, err := strconv.ParseUint(match[1], 0, 16)
			if err == nil {
				hasRet = true
				ret = uint16(errno)
			} else {
				return nil, false, 0, true, newExpr, err
			}
		}
	}
	return nil, hasRet, ret, false, newExpr, nil
}

func (p *parser) parseExpression(expr string) (tree.Expression, bool, uint16, error) {
	expression, hasRet, ret, done, expr, err := p.parseSpecialCases(expr)
	if done {
		return expression, hasRet, ret, err
	}

	tokens, err := tokenize(expr, func(ts, te int, data []byte) error {
		trace := "<input>:-1"
		if p.traceData != nil {
			trace = fmt.Sprintf("%s:%d", p.traceData.file, p.traceData.line)
		}
		return fmt.Errorf("unexpected token at %s:%d: '%s'", trace, ts, data[ts:te])

	})
	if err != nil {
		return nil, false, 0, err
	}
	ctx := parseContext{0, tokens, len(tokens) == 0, p}

	if ctx.atEnd {
		return nil, hasRet, ret, nil
	}

	expression, err = ctx.logicalORExpression()
	if err != nil {
		return nil, false, 0, err
	}

	if err = ctx.end(); err != nil {
		return nil, false, 0, err
	}

	return expression, hasRet, ret, nil
}

func (ctx *parseContext) genErr(exp string) error {
	found := "EOF"
	if !ctx.atEnd {
		td := ""
		if ctx.tokens[ctx.index].td != nil {
			td = " " + string(ctx.tokens[ctx.index].td)
		}
		found = fmt.Sprintf("'%s'%s", tokens[ctx.tokens[ctx.index].t], td)
	}
	return fmt.Errorf("expression is invalid. unable to parse: expected %s, found %s", exp, found)
}

func (ctx *parseContext) end() error {
	if !ctx.atEnd {
		return ctx.genErr("EOF")
	}
	return nil
}

func (ctx *parseContext) logicalORExpression() (tree.Expression, error) {
	left, e := ctx.logicalANDExpression()
	if e != nil {
		return nil, e
	}
	if ctx.next() == LOR {
		ctx.consume()
		right, e := ctx.logicalORExpression()
		if e != nil {
			return nil, e
		}
		return tree.Or{Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) logicalANDExpression() (tree.Expression, error) {
	left, e := ctx.equalityExpression()
	if e != nil {
		return nil, e
	}
	if ctx.next() == LAND {
		ctx.consume()
		right, e := ctx.logicalANDExpression()
		if e != nil {
			return nil, e
		}
		return tree.And{Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) equalityExpression() (tree.Expression, error) {
	left, e := ctx.relationalExpression()
	if e != nil {
		return nil, e
	}
	switch ctx.next() {
	case EQL, NEQ, BITSET:
		op, _ := ctx.consume()
		right, e := ctx.equalityExpression()
		if e != nil {
			return nil, e
		}
		return tree.Comparison{Op: comparisonOperator[op], Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) relationalExpression() (tree.Expression, error) {
	left, e := ctx.inclusiveORExpression()
	if e != nil {
		return nil, e
	}
	switch ctx.next() {
	case LT, GT, LTE, GTE:
		op, _ := ctx.consume()
		right, e := ctx.relationalExpression()
		if e != nil {
			return nil, e
		}
		return tree.Comparison{Op: comparisonOperator[op], Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) inclusiveORExpression() (tree.Expression, error) {
	left, e := ctx.exclusiveORExpression()
	if e != nil {
		return nil, e
	}
	if ctx.next() == OR {
		ctx.consume()
		right, e := ctx.inclusiveORExpression()
		if e != nil {
			return nil, e
		}
		return tree.Arithmetic{Op: tree.BINOR, Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) exclusiveORExpression() (tree.Expression, error) {
	left, e := ctx.andExpression()
	if e != nil {
		return nil, e
	}
	if ctx.next() == XOR {
		ctx.consume()
		right, e := ctx.exclusiveORExpression()
		if e != nil {
			return nil, e
		}

		return tree.Arithmetic{Op: tree.BINXOR, Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) andExpression() (tree.Expression, error) {
	left, e := ctx.shiftExpression()
	if e != nil {
		return nil, e
	}
	if ctx.next() == AND {
		ctx.consume()
		right, e := ctx.andExpression()
		if e != nil {
			return nil, e
		}
		return tree.Arithmetic{Op: tree.BINAND, Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) shiftExpression() (tree.Expression, error) {
	left, e := ctx.additiveExpression()
	if e != nil {
		return nil, e
	}
	switch ctx.next() {
	case LSH, RSH:
		op, _ := ctx.consume()
		right, e := ctx.shiftExpression()
		if e != nil {
			return nil, e
		}
		return tree.Arithmetic{Op: shiftOperator[op], Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) additiveExpression() (tree.Expression, error) {
	left, e := ctx.multiplicativeExpression()
	if e != nil {
		return nil, e
	}
	switch ctx.next() {
	case ADD, SUB:
		op, _ := ctx.consume()
		right, e := ctx.additiveExpression()
		if e != nil {
			return nil, e
		}
		return tree.Arithmetic{Op: addOperator[op], Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) multiplicativeExpression() (tree.Expression, error) {
	left, e := ctx.unaryExpression()
	if e != nil {
		return nil, e
	}
	switch ctx.next() {
	case MUL, DIV, MOD:
		op, _ := ctx.consume()
		right, e := ctx.multiplicativeExpression()
		if e != nil {
			return nil, e
		}
		return tree.Arithmetic{Op: multOperator[op], Left: left, Right: right}, nil
	}
	return left, nil
}

func (ctx *parseContext) unaryExpression() (tree.Expression, error) {
	switch ctx.next() {
	case INV:
		ctx.consume()
		left, e := ctx.unaryExpression()
		if e != nil {
			return nil, e
		}
		return tree.BinaryNegation{left}, nil
	case NOT:
		ctx.consume()
		left, e := ctx.unaryExpression()
		if e != nil {
			return nil, e
		}
		return tree.Negation{left}, nil
	}
	return ctx.primary()
}

func (ctx *parseContext) collectArgs() ([]tree.Any, error) {
	args := []tree.Any{}
	for ctx.next() != RPAREN {
		ctx.consume() //consume the opening paren, or last comma
		res, e := ctx.logicalORExpression()
		if e != nil {
			return nil, e
		}
		args = append(args, res)
		switch ctx.next() {
		case RPAREN, COMMA:
			// Do nothing here
		default:
			return nil, ctx.genErr("')' or ','")
		}
	}
	ctx.consume()
	return args, nil
}

func (ctx *parseContext) collectNumerics() ([]tree.Numeric, error) {
	args := []tree.Numeric{}
	for ctx.next() != RPAREN {
		ctx.consume() //consume the opening paren, or last comma
		res, e := ctx.logicalORExpression()
		if e != nil {
			return nil, e
		}
		args = append(args, res)
		switch ctx.next() {
		case RPAREN, COMMA:
			// Do nothing here
		default:
			return nil, ctx.genErr("')' or ','")
		}
	}
	ctx.consume()
	return args, nil
}

func (ctx *parseContext) primary() (tree.Expression, error) {
	switch ctx.next() {
	case LPAREN:
		ctx.consume()
		val, e := ctx.logicalORExpression()
		if e != nil {
			return nil, e
		}
		op, _ := ctx.consume()
		if op != RPAREN {
			return nil, ctx.genErr("')'")
		}
		return val, nil
	case ARG:
		_, data := ctx.consume()
		tp := tree.Full
		pref := strings.TrimPrefix(string(data), "arg")
		if strings.HasPrefix(pref, "H") {
			tp = tree.Hi
			pref = strings.TrimPrefix(pref, "H")
		}
		if strings.HasPrefix(pref, "L") {
			tp = tree.Low
			pref = strings.TrimPrefix(pref, "L")
		}
		val, _ := strconv.Atoi(pref)
		// This should never error out
		return tree.Argument{Index: val, Type: tp}, nil
	case IDENT:
		_, data := ctx.consume()
		if ctx.next() == LPAREN {
			args, e := ctx.collectArgs()
			if e != nil {
				return nil, e
			}
			return tree.Call{Name: string(data), Args: args}, nil
		}
		return tree.Variable{string(data)}, nil
	case IN, NOTIN:
		op, _ := ctx.consume()
		if ctx.next() == LPAREN {
			all, e := ctx.collectNumerics()
			if e != nil {
				return nil, e
			}
			return tree.Inclusion{Positive: op == IN, Left: all[0], Rights: all[1:]}, nil
		}
		return nil, ctx.genErr("'('")
	case INT:
		_, data := ctx.consume()
		// This error is ignored since unless the tokenizer is completely broken, this should never break
		val, _ := strconv.ParseUint(string(data), 0, 64)
		return tree.NumericLiteral{uint64(val)}, nil
	case TRUE:
		ctx.consume()
		return tree.BooleanLiteral{true}, nil
	case FALSE:
		ctx.consume()
		return tree.BooleanLiteral{false}, nil
	case EOF:
		return nil, errors.New("unexpected end of line")
	}

	return nil, ctx.genErr("primary expression")
}
