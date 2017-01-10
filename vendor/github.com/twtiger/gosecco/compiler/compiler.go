package compiler

import (
	"errors"
	"fmt"
	"sort"
	"syscall"

	"github.com/twtiger/gosecco/constants"
	"github.com/twtiger/gosecco/tree"

	"golang.org/x/sys/unix"
)

// TODO: compare go-seccomp and gosecco policy evaluation

// Compile will take a parsed policy and generate an optimized sock filter for that policy
// The policy is assumed to have been unified and simplified before compilation starts -
// no unresolved variables or calls should exist in the policy.
func Compile(policy tree.Policy) ([]unix.SockFilter, error) {
	c := createCompilerContext()
	return c.compile(policy)
}

type label string

type compilerContext struct {
	result                                          []unix.SockFilter
	currentlyLoaded                                 int
	stackTop                                        uint32
	jts                                             *jumpMap
	jfs                                             *jumpMap
	uconds                                          *jumpMap
	labels                                          *labelMap
	labelCounter                                    int
	defaultPositive, defaultNegative, defaultPolicy string
	actions                                         map[string]label
	maxJumpSize                                     int // this will always be 0xFF in production, but can be injected for testing.
	currentlyCompilingSyscall                       string
	currentlyCompilingExpression                    tree.Expression
}

func createCompilerContext() *compilerContext {
	return &compilerContext{
		jts:             createJumpMap(),
		jfs:             createJumpMap(),
		uconds:          createJumpMap(),
		labels:          createLabelMap(),
		actions:         make(map[string]label),
		maxJumpSize:     255,
		currentlyLoaded: -1,
	}
}

// setDefaults sets the defaults for the compiler - it should always be called before compiling anything
func (c *compilerContext) setDefaults(positive, negative, policy string) {
	if positive == "" || negative == "" || policy == "" {
		panic("The defaults should never be empty. This is a programmer error.")
	}

	c.defaultPositive = positive
	c.defaultNegative = negative
	c.defaultPolicy = policy
}

func (c *compilerContext) getOrCreateAction(action string) label {
	l, lExists := c.actions[action]

	if lExists {
		return l
	}

	actionLabel := c.newLabel()
	c.actions[action] = actionLabel
	return actionLabel
}

func (c *compilerContext) sortedActions() []string {
	actionOrder := []string{}

	for k := range c.actions {
		actionOrder = append(actionOrder, k)
	}

	sort.Strings(actionOrder)

	return actionOrder
}

func (c *compilerContext) compile(policy tree.Policy) ([]unix.SockFilter, error) {
	c.setDefaults(policy.DefaultPositiveAction, policy.DefaultNegativeAction, policy.DefaultPolicyAction)
	c.compileAuditArchCheck(policy.ActionOnAuditFailure)
	c.compileX32ABICheck(policy.ActionOnX32)

	for _, r := range policy.Rules {
		if err := c.compileRule(r); err != nil {
			return nil, err
		}
	}

	c.unconditionalJumpTo(c.getOrCreateAction(c.defaultPolicy))

	for _, k := range c.sortedActions() {
		c.labelHere(c.actions[k])
		action, err := actionDescriptionToK(k)
		if err != nil {
			return nil, err
		}
		c.op(OP_RET_K, action)
	}

	c.optimizeCode()

	c.fixupJumps()

	return c.result, nil
}

func (c *compilerContext) loadAt(pos uint32) {
	if c.currentlyLoaded != int(pos) {
		c.op(OP_LOAD, pos)
		c.currentlyLoaded = int(pos)
	}
}

func (c *compilerContext) loadLiteral(lit uint32) {
	c.op(OP_LOAD_VAL, lit)
	c.currentlyLoaded = -1
}

const syscallNameIndex = 0

func (c *compilerContext) loadCurrentSyscall() {
	c.loadAt(syscallNameIndex)
}

func (c *compilerContext) checkCorrectSyscall(name string, next label) {
	sys, ok := constants.GetSyscall(name)
	if !ok {
		panic("This shouldn't happen - analyzer should have caught it before compiler tries to compile it")
	}

	c.loadCurrentSyscall()
	goesNowhere := c.newLabel()
	c.opWithJumps(OP_JEQ_K, sys, goesNowhere, next)
	c.labelHere(goesNowhere)
}

func (c *compilerContext) compileRule(r *tree.Rule) error {
	next := c.newLabel()

	pos, neg := c.compileActions(r.PositiveAction, r.NegativeAction)

	c.checkCorrectSyscall(r.Name, next)

	// These are useful for debugging and helpful error messages
	c.currentlyCompilingSyscall = r.Name
	c.currentlyCompilingExpression = r.Body

	if err := c.compileExpression(r.Body, pos, neg); err != nil {
		return err
	}

	c.labelHere(next)

	return nil
}

func (c *compilerContext) compileActions(positiveAction string, negativeAction string) (label, label) {
	if positiveAction == "" {
		positiveAction = c.defaultPositive
	}
	posActionLabel := c.getOrCreateAction(positiveAction)

	if negativeAction == "" {
		negativeAction = c.defaultNegative
	}
	negActionLabel := c.getOrCreateAction(negativeAction)

	return posActionLabel, negActionLabel
}

func (c *compilerContext) op(code uint16, k uint32) {
	c.result = append(c.result, unix.SockFilter{
		Code: code,
		Jt:   0,
		Jf:   0,
		K:    k,
	})
}

func (c *compilerContext) compileExpression(x tree.Expression, pos, neg label) error {
	return compileBoolean(c, x, true, pos, neg)
}

func (c *compilerContext) newLabel() label {
	result := fmt.Sprintf("generatedLabel%03d", c.labelCounter)
	c.labelCounter++
	return label(result)
}

func (c *compilerContext) registerJumps(index int, jt, jf label) {
	c.jts.registerJump(jt, index)
	c.jfs.registerJump(jf, index)
}

func (c *compilerContext) labelHere(l label) {
	c.labels.addLabelAt(l, len(c.result))
}

func (c *compilerContext) unconditionalJumpTo(to label) {
	index := len(c.result)
	c.result = append(c.result, unix.SockFilter{
		Code: OP_JMP_K,
		Jt:   0,
		Jf:   0,
		K:    0,
	})
	c.uconds.registerJump(to, index)
}

func (c *compilerContext) opWithJumps(code uint16, k uint32, jt, jf label) {
	index := len(c.result)
	c.registerJumps(index, jt, jf)
	c.result = append(c.result, unix.SockFilter{
		Code: code,
		Jt:   0,
		Jf:   0,
		K:    k,
	})
}

func (c *compilerContext) jumpOnEq(val uint32, jt, jf label) {
	c.opWithJumps(OP_JEQ_K, val, jt, jf)
}

func (c *compilerContext) jumpIfBitSet(val uint32, jt, jf label) {
	c.opWithJumps(OP_JSET_K, val, jt, jf)
}

func (c *compilerContext) pushAToStack() error {
	if c.stackTop >= syscall.BPF_MEMWORDS {
		return errors.New("the expression is too complicated to compile. Please refer to the language documentation")
	}

	c.op(OP_STORE, c.stackTop)
	c.stackTop++
	return nil
}

func (c *compilerContext) popStackToX() error {
	if c.stackTop == 0 {
		return errors.New("popping from empty stack - this is likely a programmer error")
	}
	c.stackTop--
	c.op(OP_LOAD_MEM_X, c.stackTop)
	return nil
}
