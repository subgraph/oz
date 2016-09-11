package compiler

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// This file contains space for implementing peephole optimization
// For now it doesn't do anything, but it could once we know what
// patterns show up

// One common pattern we likely will want to fix is:
// [ST n, LDX n]
// and rewrite it into [TAX]

// We might also see [LD_IMM v, ST n, LDX n]
// This should be rewritten into [LDX_IMM v]

// Some patterns look amenable to optimization but in practice won't be
// - it's important that we are wary of trying to fix up jumps too much.

// We will see a lot of things like [LD_IMM v, ST n, ... LDX n, <op>]
// These, both arithmetic and comparison ones should be rewritten to use the _K
// variants. However, these is easier done in the actual compiler at the moment
// These optimizations should also check if they operate on commutative
// operators and try to put the constant to the right, where K can be used.

func (c *compilerContext) optimizeCode() {
	// We run optimizations over and over until we can't apply anymore
	for c.optimizeCycle() {
	}
}

func (c *compilerContext) optimizeCycle() bool {
	optimized := false
	index := 0

	// Do not pull out the length calculation here, since the length
	// can change during optimization
	for index < len(c.result) {
		res := c.optimizeAt(index)
		if res {
			optimized = true
		} else {
			index++
		}
	}
	return optimized
}

type optimizer func(*compilerContext, int) bool

var optimizers = []optimizer{
	jumpAfterConditionalJumpOptimizer,
	loadAndCompareWithImmediate,
	loadAndPerformArithmeticWithImmediateOptimizer,
}

func (c *compilerContext) optimizeAt(i int) bool {
	optimized := false
	for _, o := range optimizers {
		if o(c, i) {
			optimized = true
		}
	}
	return optimized
}

func isJump(s unix.SockFilter) bool {
	return bpfClass(s.Code) == syscall.BPF_JMP
}

func isConditionalJump(s unix.SockFilter) bool {
	return isJump(s) && bpfOp(s.Code) != syscall.BPF_JA
}

func isUnconditionalJump(s unix.SockFilter) bool {
	return isJump(s) && bpfOp(s.Code) == syscall.BPF_JA
}

// hasJumpTarget will return true if the conditional jump given has at least one of
// its target being the potential target.
func hasJumpTarget(c *compilerContext, conditionalIndex, potentialTarget int) bool {
	return c.jts.hasJumpTarget(c, conditionalIndex, potentialTarget) ||
		c.jfs.hasJumpTarget(c, conditionalIndex, potentialTarget)
}

// hasOnlyJumpFrom will make sure that the given jump target is only jumped to from the
// given expected from location - this will return false if the expectedFrom is a
// conditional jump where both conditions point to the jump target
func hasOnlyJumpFrom(c *compilerContext, jumpTarget, expectedFrom int) bool {
	return (c.jts.countJumpsFrom(c, jumpTarget, expectedFrom)+
		c.jfs.countJumpsFrom(c, jumpTarget, expectedFrom)+
		c.uconds.countJumpsFrom(c, jumpTarget, expectedFrom)) == 1 &&
		(c.jts.countJumpsFromAny(c, jumpTarget)+
			c.jfs.countJumpsFromAny(c, jumpTarget)+
			c.uconds.countJumpsFromAny(c, jumpTarget)) == 1
}

// isNotOversizedJump takes a pointer to an unconditional jump and return true if
// the jump is smaller than the max jump size.
func isNotOversizedJump(c *compilerContext, jumpPoint int) bool {
	return !c.uconds.jumpSizeIsOversized(c, jumpPoint)
}

func redirectJumpOf(c *compilerContext, ucond, cond int) {
	newJumpTarget := c.uconds.jumpTargetOf(ucond)
	c.uconds.removeJumpTarget(ucond)
	var sourceJm *jumpMap

	if c.jts.countJumpsFrom(c, ucond, cond) == 1 {
		sourceJm = c.jts
	} else if c.jfs.countJumpsFrom(c, ucond, cond) == 1 {
		sourceJm = c.jfs
	} else {
		panic(fmt.Sprintf("No jumps to redirect (programmer error): ucond: %d cond: %d\n%#v\n%#v\n", ucond, cond, c.jts, c.labels))
	}

	oldLabel := c.labels.labelsAt(ucond)[0]
	sourceJm.redirectJump(oldLabel, newJumpTarget)
	c.labels.removeLabel(oldLabel)
}

func (c *compilerContext) removeInstructionAt(index int) {
	c.result = append(c.result[:index], c.result[index+1:]...)
}

// jumpAfterConditionalJumpOptimizer will optimize situations where a JMP instruction
// directly follows a conditional jump where one of the arms of the conditional jump
// is zero. It will make sure that no other jump points end up on the specific JMP instruction
// before removing it. It will also make sure the resulting jump is not too large.
// An example of a fragment that would be changed would be this:
//    jeq_k	00	01	3D
//    jmp	13
// This can be optimized to:
//    jeq_k	13	00	3D
func jumpAfterConditionalJumpOptimizer(c *compilerContext, ix int) bool {
	optimized := false

	if ix+1 < len(c.result) {
		oneIndex, twoIndex := ix, ix+1
		one, two := c.result[oneIndex], c.result[twoIndex]
		if isConditionalJump(one) &&
			isUnconditionalJump(two) &&
			hasJumpTarget(c, oneIndex, twoIndex) &&
			hasOnlyJumpFrom(c, twoIndex, oneIndex) &&
			isNotOversizedJump(c, twoIndex) {

			redirectJumpOf(c, twoIndex, oneIndex)
			c.shiftJumpsBy(oneIndex+1, -1)
			c.removeInstructionAt(twoIndex)

			optimized = true
		}
	}

	return optimized
}

func isImmediateLoad(s unix.SockFilter) bool {
	return bpfClass(s.Code) == syscall.BPF_LD &&
		bpfMode(s.Code) == syscall.BPF_IMM
}

func isStore(s unix.SockFilter) bool {
	return bpfClass(s.Code) == syscall.BPF_ST
}

func isMemoryLoadIntoX(s unix.SockFilter) bool {
	return bpfClass(s.Code) == syscall.BPF_LDX &&
		bpfMode(s.Code) == syscall.BPF_MEM
}

func hasX(s unix.SockFilter) bool {
	return bpfSrc(s.Code) == syscall.BPF_X
}

func isArithmeticWithX(s unix.SockFilter) bool {
	return bpfClass(s.Code) == syscall.BPF_ALU && hasX(s)
}

func isConditionalJumpWithX(s unix.SockFilter) bool {
	return isConditionalJump(s) && hasX(s)
}

func storeLocationOf(s unix.SockFilter) uint32 {
	return s.K
}

func loadLocationOf(s unix.SockFilter) uint32 {
	return s.K
}

func sameStorageLocation(store, load unix.SockFilter) bool {
	return storeLocationOf(store) == loadLocationOf(load)
}

// loadAndCompareWithImmediate will remove a pattern that is a side effect of our
// stack based stupid compiler - it will always push stuff on the stack and then
// pop it into X to compare. This is very ineffecient and this optimizer will fix
// that.
//
// Example:
//     ld_imm	0
//     st	0
//     ld_abs	18
//     ldx_mem	0
//     jeq_x	4A	4B
// This is not great.
// It can be reduced to:
//     ld_abs  18
//     jeq_k   4A   4B   0
func loadAndCompareWithImmediate(c *compilerContext, ix int) bool {
	return loadStoreOptimizer(c, ix, isConditionalJumpWithX)
}

func loadAndPerformArithmeticWithImmediateOptimizer(c *compilerContext, ix int) bool {
	return loadStoreOptimizer(c, ix, isArithmeticWithX)
}

func loadStoreOptimizer(c *compilerContext, ix int, f func(unix.SockFilter) bool) bool {
	optimized := false

	if ix+4 < len(c.result) {
		oneIndex, twoIndex, fourIndex, fiveIndex := ix, ix+1, ix+3, ix+4
		one, two, four, five := c.result[oneIndex], c.result[twoIndex], c.result[fourIndex], c.result[fiveIndex]
		if isImmediateLoad(one) &&
			isStore(two) &&
			isMemoryLoadIntoX(four) &&
			f(five) &&
			sameStorageLocation(two, four) {
			c.result[fiveIndex].K = one.K
			c.result[fiveIndex].Code = replaceXWithKIn(c.result[fiveIndex].Code)

			c.shiftJumpsBy(oneIndex, -1)
			c.removeInstructionAt(oneIndex)

			c.shiftJumpsBy(oneIndex, -1)
			c.removeInstructionAt(oneIndex)

			c.shiftJumpsBy(oneIndex+1, -1)
			c.removeInstructionAt(oneIndex + 1)

			optimized = true
		}
	}

	return optimized
}

func replaceXWithKIn(code uint16) uint16 {
	return (code & ^uint16(syscall.BPF_X)) | uint16(syscall.BPF_K)
}
