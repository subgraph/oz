package seccomp

import (
	"fmt"
	"syscall"
)

func render_mmap(pid int, args RegisterArgs) (string, error) {

	protflags := map[uint]string{
		syscall.PROT_READ:  "PROT_READ",
		syscall.PROT_WRITE: "PROT_WRITE",
		syscall.PROT_EXEC:  "PROT_EXEC",
	}

	mmapflags := map[uint]string{
		syscall.MAP_32BIT:      "MAP_32BIT",
		syscall.MAP_ANONYMOUS:  "MAP_ANONYMOUS",
		syscall.MAP_EXECUTABLE: "MAP_EXECUTABLE",
		syscall.MAP_FILE:       "MAP_FILE",
		syscall.MAP_FIXED:      "MAP_FIXED",
		syscall.MAP_GROWSDOWN:  "MAP_GROWSDOWN",
		syscall.MAP_HUGETLB:    "MAP_HUGETLB",
		//		MAP_HUGE_2MB : "MAP_HUGE_2MB",
		//		HUGE_1GB : "MAP_HUGE_1GB",
		syscall.MAP_LOCKED:    "MAP_LOCKED",
		syscall.MAP_NONBLOCK:  "MAP_NONBLOCK",
		syscall.MAP_NORESERVE: "MAP_NORESERVE",
		syscall.MAP_POPULATE:  "MAP_POPULATE",
		syscall.MAP_STACK:     "MAP_STACK",
		//		MAP_UNINITIALIZED : "MAP_UNINITIALIZED",
	}

	mode := args[2]
	mmapflagsval := args[3]

	protflagstr := ""
	mmapflagstr := ""

	if mode == syscall.PROT_NONE {
		protflagstr = "PROT_NONE"
	} else {
		protflagstr = renderFlags(protflags, uint(mode))
	}


	if (mmapflagsval & syscall.MAP_PRIVATE) == syscall.MAP_PRIVATE {
		mmapflagstr += "MAP_PRIVATE"
	} else if (mmapflagsval & syscall.MAP_SHARED) == syscall.MAP_SHARED {
		mmapflagstr += "MAP_SHARED"
	}

	tmp := renderFlags(mmapflags, uint(mmapflagsval))
	if (tmp != "") {
		mmapflagstr += "|"
		mmapflagstr += tmp
	}

	callrep := fmt.Sprintf("mmap(0x%X, %d, %s, %s, %d, %d)", uintptr(args[0]), args[1], protflagstr, mmapflagstr, args[4], args[5])

	return fmt.Sprintf("==============================================\nseccomp hit on sandbox pid %v (%v) syscall %v (%v): \n\n%s\nI ==============================================\n\n", pid, getProcessCmdLine(pid), "mmap", syscall.SYS_MMAP, callrep), nil
}
