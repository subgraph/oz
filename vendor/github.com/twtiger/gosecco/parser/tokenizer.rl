package parser

import "fmt"

%%{
    machine gosecco_tokenizer;

    BIN_DIGIT = [01] ;
    OCT_DIGIT = [0-7] ;

    INTBIN = "0b"i BIN_DIGIT+ ;
    INTHEX = "0x"i xdigit+ ;
    INTOCT = "0" OCT_DIGIT+ ;
    INTDEC = ( "0" | ( [1-9] digit* ) ) ;

    SPACES = [ \t] ;

    IDENT_CHAR = "_" | alnum ;

    ARG = "arg" [HL]? [0-5] ;

    IDENT = [_a-zA-Z] IDENT_CHAR* ;

    main := |*
      ARG     => {f(ARG,   data[ts:te])};

      "in"i  => {f(IN, nil)};
      "notIn"i  => {f(NOTIN, nil)};

      "true"i  => {f(TRUE, nil)};
      "false"i => {f(FALSE, nil)};

      IDENT   => {f(IDENT, data[ts:te])};

      INTHEX  => {f(INT,   data[ts:te])};
      INTOCT  => {f(INT,   data[ts:te])};
      INTBIN  => {f(INT,   data[ts:te])};
      INTDEC  => {f(INT,   data[ts:te])};

      "+" => {f(ADD, nil)};
      "-" => {f(SUB, nil)};
      "*" => {f(MUL, nil)};
      "/" => {f(DIV, nil)};
      "%" => {f(MOD, nil)};

      "&?" => {f(BITSET, nil)};

      "&&" => {f(LAND, nil)};
      "||" => {f(LOR, nil)};

      "&" => {f(AND, nil)};
      "|" => {f(OR, nil)};
      "^" => {f(XOR, nil)};
      "<<" => {f(LSH, nil)};
      ">>" => {f(RSH, nil)};
      "~" => {f(INV, nil)};

      "==" => {f(EQL, nil)};
      "<=" => {f(LTE, nil)};
      ">=" => {f(GTE, nil)};
      "<" => {f(LT, nil)};
      ">" => {f(GT, nil)};
      "!=" => {f(NEQ, nil)};
      "!" => {f(NOT, nil)};

      "(" => {f(LPAREN, nil)};
      "[" => {f(LBRACK, nil)};
      ")" => {f(RPAREN, nil)};
      "]" => {f(RBRACK, nil)};

      "," => {f(COMMA, nil)};

      SPACES+;

      any => {return tokenError(ts, te, data)};
    *|;
}%%

%% write data;

func tokenizeRaw(data []byte, f func(token, []byte), tokenError func(int, int, []byte) error) error {
     var cs, act int
     p, pe := 0, len(data)
     ts, te := 0, 0
     eof := pe

     %% write init;
     %% write exec;

     return nil
}
