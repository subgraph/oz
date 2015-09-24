package seccomp

import (
	"fmt"
	"syscall"
)

func render_mprotect(pid int, args RegisterArgs) (string, error) {

	flags := map[uint]string{
		syscall.PROT_NONE:  "PROT_NONE",
		syscall.PROT_READ:  "PROT_READ",
		syscall.PROT_WRITE: "PROT_WRITE",
		syscall.PROT_EXEC:  "PROT_EXEC",
	}

	mode := args[2]
	flagstr := ""

	if mode == syscall.PROT_NONE {
		flagstr = "PROT_NONE"
	} else {
		flagstr = renderFlags(flags, uint(mode))
	}
	callrep := fmt.Sprintf("mprotect(0x%X, %d, %s)", uintptr(args[0]), args[1], flagstr)

	return fmt.Sprintf("==============================================\nseccomp hit on sandbox pid %v (%v) syscall %v (%v): \n\n%s\nI ==============================================\n\n", pid, getProcessCmdLine(pid), "mprotect", syscall.SYS_MPROTECT, callrep), nil
}
