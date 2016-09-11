package compiler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/twtiger/gosecco/constants"
)

const (
	SECCOMP_RET_KILL  = uint32(0x00000000) /* kill the task immediately */
	SECCOMP_RET_TRAP  = uint32(0x00030000) /* disallow and force a SIGSYS */
	SECCOMP_RET_ERRNO = uint32(0x00050000) /* returns an errno */
	SECCOMP_RET_TRACE = uint32(0x7ff00000) /* pass to a tracer or disallow */
	SECCOMP_RET_ALLOW = uint32(0x7fff0000) /* allow */
)

// actionDescriptionToK turns string specifications of return actions into compiled values acceptable for the compiler to insert
func actionDescriptionToK(v string) (action uint32, err error) {
	switch strings.ToLower(v) {
	case "trap":
		return SECCOMP_RET_TRAP, nil
	case "kill":
		return SECCOMP_RET_KILL, nil
	case "allow":
		return SECCOMP_RET_ALLOW, nil
	case "trace":
		return SECCOMP_RET_TRACE, nil
	}

	if res, err := strconv.ParseUint(v, 0, 16); err == nil {
		return SECCOMP_RET_ERRNO | uint32(res), nil
	}

	if res, ok := constants.GetError(v); ok {
		return SECCOMP_RET_ERRNO | res, nil
	}

	return 0, fmt.Errorf("Invalid return action '%s'", v)
}
