package tree

// Transformer is something that can transform expressions
type Transformer interface {
	Visitor
	Transform(Expression) Expression
}

// EmptyTransformer does nothing - it returns the same tree as given
// It can be useful as the base for other transformers
type EmptyTransformer struct {
	Result   Expression
	RealSelf Transformer
}

// Transform implements Transformer
func (s *EmptyTransformer) Transform(inp Expression) Expression {
	inp.Accept(s.RealSelf)
	return s.Result
}

// AcceptAnd implements Visitor
func (s *EmptyTransformer) AcceptAnd(v And) {
	s.Result = And{
		Left:  s.Transform(v.Left),
		Right: s.Transform(v.Right),
	}
}

// AcceptArgument implements Visitor
func (s *EmptyTransformer) AcceptArgument(v Argument) {
	s.Result = v
}

// AcceptArithmetic implements Visitor
func (s *EmptyTransformer) AcceptArithmetic(v Arithmetic) {
	s.Result = Arithmetic{
		Op:    v.Op,
		Left:  s.Transform(v.Left),
		Right: s.Transform(v.Right),
	}
}

// AcceptBinaryNegation implements Visitor
func (s *EmptyTransformer) AcceptBinaryNegation(v BinaryNegation) {
	s.Result = BinaryNegation{s.Transform(v.Operand)}
}

// AcceptBooleanLiteral implements Visitor
func (s *EmptyTransformer) AcceptBooleanLiteral(v BooleanLiteral) {
	s.Result = v
}

// AcceptCall implements Visitor
func (s *EmptyTransformer) AcceptCall(v Call) {
	result := make([]Any, len(v.Args))
	for ix, v2 := range v.Args {
		result[ix] = s.Transform(v2)
	}
	s.Result = Call{Name: v.Name, Args: result}
}

// AcceptComparison implements Visitor
func (s *EmptyTransformer) AcceptComparison(v Comparison) {
	s.Result = Comparison{
		Op:    v.Op,
		Left:  s.Transform(v.Left),
		Right: s.Transform(v.Right),
	}
}

// AcceptInclusion implements Visitor
func (s *EmptyTransformer) AcceptInclusion(v Inclusion) {
	result := make([]Numeric, len(v.Rights))
	for ix, v2 := range v.Rights {
		result[ix] = s.Transform(v2)
	}
	s.Result = Inclusion{
		Positive: v.Positive,
		Left:     s.Transform(v.Left),
		Rights:   result}
}

// AcceptNegation implements Visitor
func (s *EmptyTransformer) AcceptNegation(v Negation) {
	s.Result = Negation{s.Transform(v.Operand)}
}

// AcceptNumericLiteral implements Visitor
func (s *EmptyTransformer) AcceptNumericLiteral(v NumericLiteral) {
	s.Result = v
}

// AcceptOr implements Visitor
func (s *EmptyTransformer) AcceptOr(v Or) {
	s.Result = Or{
		Left:  s.Transform(v.Left),
		Right: s.Transform(v.Right),
	}
}

// AcceptVariable implements Visitor
func (s *EmptyTransformer) AcceptVariable(v Variable) {
	s.Result = v
}
