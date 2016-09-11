package parser

type parseContext struct {
	index  int
	tokens []tokenData
	atEnd  bool
	parser *parser
}

func (ctx *parseContext) next() token {
	if ctx.atEnd {
		return EOF
	}
	return ctx.tokens[ctx.index].t
}

func (ctx *parseContext) advance() {
	ctx.index++
	if ctx.index >= len(ctx.tokens) {
		ctx.atEnd = true
	}
}

func (ctx *parseContext) consume() (token, []byte) {
	if ctx.atEnd {
		return EOF, nil
	}
	res := ctx.tokens[ctx.index]
	ctx.advance()
	return res.t, res.td
}
