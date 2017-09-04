package unifier

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/twtiger/gosecco/tree"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type UnifierSuite struct{}

var _ = Suite(&UnifierSuite{})

func (s *UnifierSuite) Test_Unify_withNothingToUnify(c *C) {
	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{},
	}

	output, _ := Unify(input, nil, "", "", "")

	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 0)
}

func (s *UnifierSuite) Test_Unify_withRuleToUnify(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.NumericLiteral{42}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")

	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(*(output.Rules[0]), Equals, rule)
}

func (s *UnifierSuite) Test_Unify_withRuleAndMacroThatDoesntUnify(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.NumericLiteral{42}},
	}

	macro := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{1},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
			macro,
		},
	}

	output, _ := Unify(input, nil, "", "", "")

	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["var1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(*(output.Rules[0]), DeepEquals, rule)
}

func (s *UnifierSuite) Test_Unify_withRuleAndMacroToActuallyUnify(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.Variable{"var1"}},
	}

	macro := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{1},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["var1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 1)")
}

func (s *UnifierSuite) Test_Unify_orExpression(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Or{Left: tree.Argument{Index: 0}, Right: tree.Variable{"var1"}},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{1},
	}

	macro2 := tree.Macro{
		Name: "var2",
		Body: tree.NumericLiteral{2},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro1,
			macro2,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 2)
	c.Assert(output.Macros["var1"], DeepEquals, macro1)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(or arg0 1)")
}

func (s *UnifierSuite) Test_Unify_withAndExpressione(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.And{Left: tree.Argument{Index: 0}, Right: tree.Variable{"var1"}},
	}

	macro := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{1},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["var1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(and arg0 1)")
}

func (s *UnifierSuite) Test_Unify_withArithmeticExpression(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Arithmetic{Left: tree.Argument{Index: 0}, Op: tree.PLUS, Right: tree.Variable{"var1"}},
	}

	macro := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{1},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["var1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(plus arg0 1)")
}

func (s *UnifierSuite) Test_Unify_withInclusionExpression(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Inclusion{Positive: true,
			Left:   tree.Argument{Index: 0},
			Rights: []tree.Numeric{tree.NumericLiteral{1}, tree.Variable{"var2"}},
		},
	}

	macro := tree.Macro{
		Name: "var2",
		Body: tree.NumericLiteral{2},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["var2"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(in arg0 1 2)")
}

func (s *UnifierSuite) Test_Unify_withInclusionExpressionVariableLeft(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Inclusion{Positive: true,
			Left:   tree.Variable{"var1"},
			Rights: []tree.Numeric{tree.NumericLiteral{1}, tree.Variable{"var2"}},
		},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.Argument{Index: 0},
	}

	macro2 := tree.Macro{
		Name: "var2",
		Body: tree.NumericLiteral{2},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro1,
			macro2,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 2)
	c.Assert(output.Macros["var1"], DeepEquals, macro1)
	c.Assert(output.Macros["var2"], DeepEquals, macro2)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(in arg0 1 2)")
}

func (s *UnifierSuite) Test_Unify_withNegationExpression(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Negation{Operand: tree.Variable{"var1"}},
	}

	macro := tree.Macro{
		Name: "var1",
		Body: tree.BooleanLiteral{true},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["var1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(not true)")
}

func (s *UnifierSuite) Test_Unify_withCallExpression(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Call{Name: "compV1", Args: []tree.Any{tree.Argument{Index: 0}}},
	}

	macro := tree.Macro{
		Name:          "compV1",
		ArgumentNames: []string{"var1"},
		Body:          tree.Comparison{Left: tree.Variable{"var1"}, Op: tree.EQL, Right: tree.NumericLiteral{1}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["compV1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 1)")
}

func (s *UnifierSuite) Test_Unify_withCallExpressionWithMultipleVariables(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Call{Name: "compV1", Args: []tree.Any{tree.Argument{Index: 0}, tree.Argument{Index: 1}}},
	}

	macro := tree.Macro{
		Name:          "compV1",
		ArgumentNames: []string{"var1", "var2"},
		Body:          tree.Comparison{Left: tree.Variable{"var1"}, Op: tree.EQL, Right: tree.Variable{"var2"}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["compV1"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 arg1)")
}

func (s *UnifierSuite) Test_Unify_withCallExpressionWithPreviouslyDefinedVariables(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Call{Name: "compV1", Args: []tree.Any{tree.Argument{Index: 0}, tree.Variable{"var2"}}},
	}

	macro1 := tree.Macro{
		Name: "var2",
		Body: tree.Argument{Index: 5},
	}

	macro2 := tree.Macro{
		Name:          "compV1",
		ArgumentNames: []string{"var1", "var2"},
		Body:          tree.Comparison{Left: tree.Variable{"var1"}, Op: tree.EQL, Right: tree.Variable{"var2"}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro1,
			macro2,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(output.Macros["var2"], DeepEquals, macro1)
	c.Assert(output.Macros["compV1"], DeepEquals, macro2)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 arg5)")
}

func (s *UnifierSuite) Test_Unify_withNoVariableDefinedRaisesNoVariableDefinedError(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.Variable{"var1"}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
		},
	}

	_, error := Unify(input, nil, "", "", "")
	c.Assert(error, ErrorMatches, "Variable 'var1' is not defined")
}

func (s *UnifierSuite) Test_Unify_withCallExpressionWhereVariableIsNotDefinedRaisesVariableUndefinedError(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Call{Name: "compV1", Args: []tree.Any{tree.Argument{Index: 0}, tree.Variable{"var2"}}},
	}

	macro1 := tree.Macro{
		Name: "var5",
		Body: tree.Argument{Index: 5},
	}

	macro2 := tree.Macro{
		Name:          "compV1",
		ArgumentNames: []string{"var1", "var2"},
		Body:          tree.Comparison{Left: tree.Variable{"var1"}, Op: tree.EQL, Right: tree.Variable{"var2"}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro1,
			macro2,
			rule,
		},
	}

	_, error := Unify(input, nil, "", "", "")
	c.Assert(error, ErrorMatches, "Variable 'var2' is not defined")
}

func (s *UnifierSuite) Test_Unify_withUndefinedCallExpressionRaisesVariableUndefinedError(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Call{Name: "compV1", Args: []tree.Any{tree.Argument{Index: 0}, tree.Variable{"var2"}}},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.Argument{Index: 5},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro1,
			rule,
		},
	}

	_, error := Unify(input, nil, "", "", "")
	c.Assert(error, ErrorMatches, "Macro 'compV1' is not defined")
}

func (s *UnifierSuite) Test_Unify_withDefaultPositiveNumericActionSetsPositiveAction(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.NumericLiteral{872}},
	}

	macro := tree.Macro{
		Name: "DEFAULT_POSITIVE",
		Body: tree.NumericLiteral{42},
	}

	macro2 := tree.Macro{
		Name: "DEFAULT_POLICY",
		Body: tree.Variable{"EACCESS"},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
			macro,
			macro2,
		},
	}

	output, _ := Unify(input, nil, "allow", "kill", "")

	c.Assert(output.DefaultPositiveAction, Equals, "42")
	c.Assert(output.DefaultNegativeAction, Equals, "kill")
	c.Assert(output.DefaultPolicyAction, Equals, "EACCESS")
	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(*(output.Rules[0]), DeepEquals, rule)
}

func (s *UnifierSuite) Test_Unify_withDefaultPositiveVariableActionSetsPositiveAction(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.NumericLiteral{42}},
	}

	macro := tree.Macro{
		Name: "DEFAULT_POSITIVE",
		Body: tree.Variable{"trace"},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
			macro,
		},
	}

	output, _ := Unify(input, nil, "allow", "kill", "")

	c.Assert(output.DefaultPositiveAction, Equals, "trace")
	c.Assert(output.DefaultNegativeAction, Equals, "kill")
	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(*(output.Rules[0]), DeepEquals, rule)
}

func (s *UnifierSuite) Test_Unify_withDefaultNegativeNumericActionSetsNegativeAction(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.NumericLiteral{42}},
	}

	macro := tree.Macro{
		Name: "DEFAULT_NEGATIVE",
		Body: tree.NumericLiteral{0},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
			macro,
		},
	}

	output, _ := Unify(input, nil, "allow", "", "")

	c.Assert(output.DefaultPositiveAction, Equals, "allow")
	c.Assert(output.DefaultNegativeAction, Equals, "0")
	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(*(output.Rules[0]), DeepEquals, rule)
}

func (s *UnifierSuite) Test_Unify_withDefaultNegativeVariableActionSetsNegativeAction(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.NumericLiteral{42}},
	}

	macro := tree.Macro{
		Name: "DEFAULT_NEGATIVE",
		Body: tree.Variable{"kill"},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
			macro,
		},
	}

	output, _ := Unify(input, nil, "allow", "kill", "")

	c.Assert(output.DefaultPositiveAction, Equals, "allow")
	c.Assert(output.DefaultNegativeAction, Equals, "kill")
	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(*(output.Rules[0]), DeepEquals, rule)
}

func (s *UnifierSuite) Test_Unify_withDefaultNegativeNumericActionSetsNegativeActionAndOtherMacros(c *C) {

	rule := tree.Rule{
		Name: "write",
		Body: tree.Or{Left: tree.Argument{Index: 0}, Right: tree.Variable{"var1"}},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{1},
	}

	macro2 := tree.Macro{
		Name: "DEFAULT_NEGATIVE",
		Body: tree.NumericLiteral{0},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro2,
			macro1,
			rule,
		},
	}

	output, _ := Unify(input, nil, "allow", "kill", "")

	c.Assert(output.DefaultPositiveAction, Equals, "allow")
	c.Assert(output.DefaultNegativeAction, Equals, "0")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(or arg0 1)")
}

func (s *UnifierSuite) Test_Unify_earlierErrorShouldStillBeReported(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Op: tree.EQL, Left: tree.Argument{Index: 0}, Right: tree.Variable{"var1"}},
	}
	rule2 := tree.Rule{
		Name: "read",
		Body: tree.Comparison{Op: tree.EQL, Left: tree.Argument{Index: 0}, Right: tree.NumericLiteral{3}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
			rule2,
		},
	}

	_, e := Unify(input, nil, "allow", "kill", "")

	c.Assert(e, Not(IsNil))
	c.Assert(e, ErrorMatches, "Variable 'var1' is not defined")
}

func (s *UnifierSuite) Test_Unify_withMacroDefinedInSeparateFile(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Op: tree.EQL, Left: tree.Argument{Index: 0}, Right: tree.Variable{"var1"}},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.NumericLiteral{42},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
		},
	}

	otherInput := []map[string]tree.Macro{
		map[string]tree.Macro{
			"var1": macro1,
		},
	}

	output, e := Unify(input, otherInput, "allow", "kill", "")

	c.Assert(e, IsNil)
	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 42)")
}

func (s *UnifierSuite) Test_Unify_withVariableReferenceToAConstant(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Op: tree.EQL, Left: tree.Argument{Index: 0}, Right: tree.Variable{"AF_ISDN"}},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			rule,
		},
	}
	output, e := Unify(input, nil, "allow", "kill", "")

	c.Assert(e, IsNil)
	c.Assert(len(output.Macros), Equals, 0)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, fmt.Sprintf("(eq arg0 %d)", syscall.AF_ISDN))
}

func (s *UnifierSuite) Test_Unify_letsMacroTakePrecedenceOverConstant(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.Variable{"AF_ISDN"}},
	}

	macro := tree.Macro{
		Name: "AF_ISDN",
		Body: tree.NumericLiteral{1},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 1)
	c.Assert(output.Macros["AF_ISDN"], DeepEquals, macro)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 1)")
}

func (s *UnifierSuite) Test_Unify_withMoreThanOneVariableResolution(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.Variable{"var1"}},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.Variable{"var2"},
	}

	macro2 := tree.Macro{
		Name: "var2",
		Body: tree.NumericLiteral{1},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro2,
			macro1,
			rule,
		},
	}

	output, _ := Unify(input, nil, "", "", "")
	c.Assert(len(output.Macros), Equals, 2)
	c.Assert(output.Macros["var1"], DeepEquals, macro1)
	c.Assert(len(output.Rules), Equals, 1)
	c.Assert(tree.ExpressionString(output.Rules[0].Body), Equals, "(eq arg0 1)")
}

func (s *UnifierSuite) Test_Unify_generatesErrorWithAMissingDependentVariable(c *C) {
	rule := tree.Rule{
		Name: "write",
		Body: tree.Comparison{Left: tree.Argument{Index: 0}, Op: tree.EQL, Right: tree.Variable{"var1"}},
	}

	macro1 := tree.Macro{
		Name: "var1",
		Body: tree.Variable{"var2"},
	}

	input := tree.RawPolicy{
		RuleOrMacros: []interface{}{
			macro1,
			rule,
		},
	}

	_, e := Unify(input, nil, "", "", "")
	c.Assert(e, Not(IsNil))
	c.Assert(e, ErrorMatches, "Variable 'var2' is not defined")
}
