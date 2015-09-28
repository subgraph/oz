package seccomp

import (
	"fmt"
	"syscall"
)

// TODO: Get constants from C headers

const (
	MREMAP_MAYMOVE = 1
	MREMAP_FIXED   = 2
)

var protflags = map[uint]string{
	syscall.PROT_READ:  "PROT_READ",
	syscall.PROT_WRITE: "PROT_WRITE",
	syscall.PROT_EXEC:  "PROT_EXEC",
}

var mmapflags = map[uint]string{
	syscall.MAP_32BIT:      "MAP_32BIT",
	syscall.MAP_ANONYMOUS:  "MAP_ANONYMOUS",
	syscall.MAP_EXECUTABLE: "MAP_EXECUTABLE",
	syscall.MAP_FILE:       "MAP_FILE",
	syscall.MAP_FIXED:      "MAP_FIXED",
	syscall.MAP_GROWSDOWN:  "MAP_GROWSDOWN",
	syscall.MAP_HUGETLB:    "MAP_HUGETLB",
	//              MAP_HUGE_2MB : "MAP_HUGE_2MB",
	//              HUGE_1GB : "MAP_HUGE_1GB",
	syscall.MAP_LOCKED:    "MAP_LOCKED",
	syscall.MAP_NONBLOCK:  "MAP_NONBLOCK",
	syscall.MAP_NORESERVE: "MAP_NORESERVE",
	syscall.MAP_POPULATE:  "MAP_POPULATE",
	syscall.MAP_STACK:     "MAP_STACK",
	//              MAP_UNINITIALIZED : "MAP_UNINITIALIZED",
}

var mremapflags = map[uint]string{
	MREMAP_MAYMOVE: "MREMAP_MAYMOVE",
	MREMAP_FIXED:   "MREMAP_FIXED",
}

func render_mmap(pid int, args RegisterArgs) (string, error) {

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
	if tmp != "" {
		mmapflagstr += "|"
		mmapflagstr += tmp
	}

	addr := ptrtostrornull(uintptr(args[0]))
	callrep := fmt.Sprintf("mmap(%s, %d, %s, %s, %d, %d)", addr, args[1], protflagstr, mmapflagstr, int32(args[4]), args[5])
	return callrep, nil
}

func render_mremap(pid int, args RegisterArgs) (string, error) {

	oldaddr := args[0]
	oldsize := args[1]
	newsize := args[2]
	flags := args[3]
	newaddr := args[4]

	flagstr := ""
	newaddrstr := ""

	if flags == 0 {
		flagstr = "0"
	} else {
		flagstr = renderFlags(mremapflags, uint(flags))
	}

	newaddrstr = ptrtostrornull(uintptr(newaddr))

	callrep := fmt.Sprintf("mremap(0x%X, %d, %d, %s, %s)", oldaddr, oldsize, newsize, flagstr, newaddrstr)
	return callrep, nil

}

func render_mprotect(pid int, args RegisterArgs) (string, error) {

	mode := args[2]
	flagstr := ""

	if mode == syscall.PROT_NONE {
		flagstr = "PROT_NONE"
	} else {
		flagstr = renderFlags(protflags, uint(mode))
	}
	callrep := fmt.Sprintf("mprotect(0x%X, %d, %s)", uintptr(args[0]), args[1], flagstr)
	return callrep, nil
}

func ptrtostrornull(ptr uintptr) string {
	if ptr == 0 {
		return "NULL"
	} else {
		return fmt.Sprintf("0x%X", ptr)
	}
}
