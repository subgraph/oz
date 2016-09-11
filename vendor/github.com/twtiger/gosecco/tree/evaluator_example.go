package tree

// EvaluatorVisitor will generate an unambigious representation of an expression
type EvaluatorVisitor struct {
	booleanResults []bool
	numericResults []uint64
}

func (sv *EvaluatorVisitor) popNumeric() uint64 {
	last := sv.numericResults[0]
	sv.numericResults = sv.numericResults[1:]
	return last
}

func (sv *EvaluatorVisitor) pushNumeric(v uint64) {
	sv.numericResults = append([]uint64{v}, sv.numericResults...)
}

func (sv *EvaluatorVisitor) popBoolean() bool {
	last := sv.booleanResults[0]
	sv.booleanResults = sv.booleanResults[1:]
	return last
}

func (sv *EvaluatorVisitor) pushBoolean(v bool) {
	sv.booleanResults = append([]bool{v}, sv.booleanResults...)
}

// AcceptAnd implements Visitor
func (sv *EvaluatorVisitor) AcceptAnd(v And) {
	v.Left.Accept(sv)
	leftVal := sv.popBoolean()
	if leftVal {
		v.Right.Accept(sv)
	} else {
		sv.pushBoolean(false)
	}
}

// AcceptArgument implements Visitor
func (sv *EvaluatorVisitor) AcceptArgument(v Argument) {}

// AcceptArithmetic implements Visitor
func (sv *EvaluatorVisitor) AcceptArithmetic(v Arithmetic) {
	v.Left.Accept(sv)
	v.Right.Accept(sv)

	right := sv.popNumeric()
	left := sv.popNumeric()

	switch v.Op {
	case PLUS:
		sv.pushNumeric(left + right)
	case MINUS:
		sv.pushNumeric(left - right)
	case MULT:
		sv.pushNumeric(left * right)
	case DIV:
		sv.pushNumeric(left / right)
	}
}

// AcceptBinaryNegation implements Visitor
func (sv *EvaluatorVisitor) AcceptBinaryNegation(v BinaryNegation) {
	v.Operand.Accept(sv)
	sv.pushNumeric(^sv.popNumeric())
}

// AcceptBooleanLiteral implements Visitor
func (sv *EvaluatorVisitor) AcceptBooleanLiteral(v BooleanLiteral) {
	sv.pushBoolean(v.Value)
}

// AcceptCall implements Visitor
func (sv *EvaluatorVisitor) AcceptCall(v Call) {}

// AcceptComparison implements Visitor
func (sv *EvaluatorVisitor) AcceptComparison(v Comparison) {
	v.Left.Accept(sv)
	v.Right.Accept(sv)

	right := sv.popNumeric()
	left := sv.popNumeric()

	switch v.Op {
	case EQL:
		sv.pushBoolean(left == right)
	case NEQL:
		sv.pushBoolean(left != right)
	case GT:
		sv.pushBoolean(left > right)
	case GTE:
		sv.pushBoolean(left >= right)
	case LT:
		sv.pushBoolean(left < right)
	case LTE:
		sv.pushBoolean(left <= right)
	}
}

// AcceptInclusion implements Visitor
func (sv *EvaluatorVisitor) AcceptInclusion(v Inclusion) {}

// AcceptNegation implements Visitor
func (sv *EvaluatorVisitor) AcceptNegation(v Negation) {
	v.Operand.Accept(sv)
	sv.pushBoolean(!sv.popBoolean())
}

// AcceptNumericLiteral implements Visitor
func (sv *EvaluatorVisitor) AcceptNumericLiteral(v NumericLiteral) {
	sv.pushNumeric(v.Value)
}

// AcceptOr implements Visitor
func (sv *EvaluatorVisitor) AcceptOr(v Or) {
	v.Left.Accept(sv)
	leftVal := sv.popBoolean()
	if !leftVal {
		v.Right.Accept(sv)
	} else {
		sv.pushBoolean(true)
	}
}

// AcceptVariable implements Visitor
func (sv *EvaluatorVisitor) AcceptVariable(v Variable) {}
