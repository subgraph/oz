package parser

import "github.com/twtiger/gosecco/tree"

var addOperator = map[token]tree.ArithmeticType{
	ADD: tree.PLUS,
	SUB: tree.MINUS,
}

var multOperator = map[token]tree.ArithmeticType{
	MUL: tree.MULT,
	DIV: tree.DIV,
	MOD: tree.MOD,
}

var shiftOperator = map[token]tree.ArithmeticType{
	LSH: tree.LSH,
	RSH: tree.RSH,
}

var comparisonOperator = map[token]tree.ComparisonType{
	EQL:    tree.EQL,
	NEQ:    tree.NEQL,
	LT:     tree.LT,
	GT:     tree.GT,
	LTE:    tree.LTE,
	GTE:    tree.GTE,
	BITSET: tree.BITSET,
}
