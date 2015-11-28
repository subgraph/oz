package seccomp

import (
	"fmt"
	"syscall"
)

// #include "linux/prctl.h"
import "C"

var prctloptions = map[uint]string{

	syscall.PR_SET_PDEATHSIG:  "PR_SET_PDEATHSIG",
	syscall.PR_GET_PDEATHSIG:  "PR_GET_PDEATHSIG",
	syscall.PR_GET_DUMPABLE:   "PR_GET_DUMPABLE",
	syscall.PR_SET_DUMPABLE:   "PR_SET_DUMPABLE",
	syscall.PR_GET_UNALIGN:    "PR_GET_UNALIGN",
	syscall.PR_SET_UNALIGN:    "PR_SET_UNALIGN",
	syscall.PR_GET_KEEPCAPS:   "PR_GET_KEEPCAPS",
	syscall.PR_SET_KEEPCAPS:   "PR_SET_KEEPCAPS",
	syscall.PR_GET_FPEMU:      "PR_GET_FPEMU",
	syscall.PR_SET_FPEMU:      "PR_SET_FPEMU",
	syscall.PR_GET_FPEXC:      "PR_GET_FPEXC",
	syscall.PR_SET_FPEXC:      "PR_SET_FPEXC",
	syscall.PR_GET_TIMING:     "PR_GET_TIMING",
	syscall.PR_SET_TIMING:     "PR_SET_TIMING",
	syscall.PR_SET_NAME:       "PR_SET_NAME",
	syscall.PR_GET_NAME:       "PR_GET_NAME",
	syscall.PR_GET_ENDIAN:     "PR_GET_ENDIAN",
	syscall.PR_SET_ENDIAN:     "PR_SET_ENDIAN",
	syscall.PR_GET_SECCOMP:    "PR_GET_SECCOMP",
	syscall.PR_SET_SECCOMP:    "PR_SET_SECCOMP",
	syscall.PR_GET_TSC:        "PR_GET_TSC",
	syscall.PR_SET_TSC:        "PR_SET_TSC",
	syscall.PR_GET_SECUREBITS: "PR_GET_SECUREBITS",
	syscall.PR_SET_SECUREBITS: "PR_SET_SECUREBITS",
	syscall.PR_SET_TIMERSLACK: "PR_SET_TIMERSLACK",
	syscall.PR_GET_TIMERSLACK: "PR_GET_TIMERSLACK",
	syscall.PR_MCE_KILL:       "PR_MCE_KILL",
	syscall.PR_MCE_KILL_GET:   "PR_MCE_KILL_GET",
	syscall.PR_SET_PTRACER:    "PR_SET_PTRACER",
	C.PR_SET_MM:               "PR_SET_MM",
	C.PR_SET_CHILD_SUBREAPER:  "PR_SET_CHILD_SUBREAPER",
	C.PR_GET_CHILD_SUBREAPER:  "PR_GET_CHILD_SUBREAPER",
	C.PR_SET_NO_NEW_PRIVS:     "PR_SET_NO_NEW_PRIVS",
	C.PR_GET_NO_NEW_PRIVS:     "PR_GET_NO_NEW_PRIVS",
	C.PR_GET_TID_ADDRESS:      "PR_GET_TID_ADDRESS",
	C.PR_SET_THP_DISABLE:      "PR_SET_THP_DISABLE",
	C.PR_GET_THP_DISABLE:      "PR_GET_THP_DISABLE",
}

func render_prctl(pid int, args RegisterArgs) (string, error) {

	prctlstr := prctloptions[uint(args[0])]

	if prctlstr == "" {
		prctlstr = fmt.Sprintf("%d", args[0])
	}

	callrep := fmt.Sprintf("prctl(%s", prctlstr)

	switch args[0] {
	case syscall.PR_SET_NAME:
		name, err := readStringArg(pid, uintptr(args[1]))
		if err != nil {
			return "", err
		}
		callrep += fmt.Sprintf(",\"%s\"", name)
	case syscall.PR_GET_TIMING:
	case syscall.PR_GET_TIMERSLACK:
	case syscall.PR_SET_TIMERSLACK:
		callrep += fmt.Sprintf(", %n", args[1])
	case C.PR_SET_CHILD_SUBREAPER:
		callrep += fmt.Sprintf(", %n", args[1])
	case C.PR_GET_CHILD_SUBREAPER:
		callrep += fmt.Sprintf(", 0x%X", args[1])
	case C.PR_SET_NO_NEW_PRIVS:
		callrep += fmt.Sprintf(", %n", args[1])
	case C.PR_GET_NO_NEW_PRIVS:
	default:
		callrep += fmt.Sprintf(", %x, %x, %x, %x", args[1], args[2], args[3], args[4])
	}
	callrep += ")"

	return callrep, nil
}
