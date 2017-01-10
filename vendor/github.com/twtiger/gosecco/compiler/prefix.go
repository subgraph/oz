package compiler

import "github.com/twtiger/gosecco/native"

const archIndex = 4

func (c *compilerContext) compileAuditArchCheck(on string) {
	if on == "" {
		on = "kill"
	}

	failure := c.getOrCreateAction(on)
	correct := c.newLabel()

	c.loadAt(archIndex)
	c.jumpOnEq(native.AuditArch, correct, failure)
	c.labelHere(correct)
}

func (c *compilerContext) compileX32ABICheck(on string) {
	if on == "" {
		return
	}

	failure := c.getOrCreateAction(on)
	correct := c.newLabel()

	c.loadAt(syscallNameIndex)
	c.jumpIfBitSet(native.X32SyscallBit, correct, failure)
	c.labelHere(correct)
}
