package compiler

import (
	"syscall"

	. "gopkg.in/check.v1"
)

type ReturnActionsSuite struct{}

var _ = Suite(&ReturnActionsSuite{})

func assertWithError(c *C, actionName string, expAction uint32, expErr string) {
	action, err := actionDescriptionToK(actionName)

	c.Assert(action, Equals, expAction)

	if expErr == "" {
		c.Assert(err, IsNil)
	} else {
		c.Assert(err, ErrorMatches, expErr)
	}
}

func (s *ReturnActionsSuite) Test_returnTrap(c *C) {
	assertWithError(c, "Trap", SECCOMP_RET_TRAP, "")
}

func (s *ReturnActionsSuite) Test_returnKill(c *C) {
	assertWithError(c, "KILL", SECCOMP_RET_KILL, "")
}

func (s *ReturnActionsSuite) Test_returnTrace(c *C) {
	assertWithError(c, "trace", SECCOMP_RET_TRACE, "")
}

func (s *ReturnActionsSuite) Test_returnAllow(c *C) {
	assertWithError(c, "AlloW", SECCOMP_RET_ALLOW, "")
}

func (s *ReturnActionsSuite) Test_returnNumericValue(c *C) {
	assertWithError(c, "42", uint32(0x5002a), "")
}

func (s *ReturnActionsSuite) Test_returnErrName(c *C) {
	assertWithError(c, "EPFNOSUPPORT", uint32(0x50000|syscall.EPFNOSUPPORT), "")
}

func (s *ReturnActionsSuite) Test_returnUnknown(c *C) {
	assertWithError(c, "Blarg", 0, "Invalid return action 'Blarg'")
}
