package parser

type token int

// These define all legal token types
const (
	ILLEGAL token = iota

	IDENT // main
	ARG   // arg[0-5]
	INT   // 12345, 0b01010, 0xFFF, 0777

	ADD // +
	SUB // -
	MUL // *
	DIV // /
	MOD // %

	AND // &
	OR  // |
	XOR // ^
	LSH // <<
	RSH // >>
	INV // ~

	LAND // &&
	LOR  // ||

	EQL // ==
	LT  // <
	GT  // >
	NOT // !

	NEQ    // !=
	LTE    // <=
	GTE    // >=
	BITSET // &?

	LPAREN // (
	LBRACK // [
	COMMA  // ,

	RPAREN // )
	RBRACK // ]

	TRUE  // true
	FALSE // false

	IN    // in
	NOTIN //notin

	EOF
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",

	IDENT: "IDENT",
	ARG:   "ARG",
	INT:   "INT",

	ADD: "+",
	SUB: "-",
	MUL: "*",
	DIV: "/",
	MOD: "%",

	AND: "&",
	OR:  "|",
	XOR: "^",
	LSH: "<<",
	RSH: ">>",
	INV: "~",

	LAND: "&&",
	LOR:  "||",

	EQL: "==",
	LT:  "<",
	GT:  ">",
	NOT: "!",

	NEQ: "!=",
	LTE: "<=",
	GTE: ">=",

	LPAREN: "(",
	LBRACK: "[",
	COMMA:  ",",

	RPAREN: ")",
	RBRACK: "]",

	TRUE:  "TRUE",
	FALSE: "FALSE",

	IN:    "in",
	NOTIN: "notIn",
}
