package seccomp

import (
	"fmt"
	"syscall"
)

// #include "asm-generic/mman-common.h"
// #include "linux/mman.h"
import "C"
/*
const (
	MREMAP_MAYMOVE = 1
	MREMAP_FIXED   = 2
)
*/
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

var adviseflags = map[uint]string{
	C.MADV_NORMAL:       "MADV_NORMAL",
	C.MADV_SEQUENTIAL:   "MADV_SEQUENTIAL",
	C.MADV_RANDOM:       "MADV_RANDOM",
	C.MADV_WILLNEED:     "MADV_WILLNEED",
	C.MADV_DONTNEED:     "MADV_DONTNEED",
	C.MADV_REMOVE:       "MADV_REMOVE",
	C.MADV_DONTFORK:     "MADV_DONTFORK",
	C.MADV_DOFORK:       "MADV_DOFORK",
	C.MADV_HWPOISON:     "MADV_HWPOISON",
	C.MADV_SOFT_OFFLINE: "MADV_SOFTOFFLINE",
	C.MADV_MERGEABLE:    "MADV_MERGEABLE",
	C.MADV_UNMERGEABLE:  "MADV_UNREMERGEABLE",
	C.MADV_HUGEPAGE:     "MADV_HUGEPAGE",
	C.MADV_NOHUGEPAGE:   "MADV_NOHUGEPAGE",
	C.MADV_DONTDUMP:     "MADV_DONTDUMP",
	C.MADV_DODUMP:       "MADV_DODUMP",
}

var mremapflags = map[uint]string{
	C.MREMAP_MAYMOVE: "MREMAP_MAYMOVE",
	C.MREMAP_FIXED:   "MREMAP_FIXED",
}

func render_madvise(pid int, args RegisterArgs) (string, error) {

	addr := args[0]
	size := uint(args[1])
	advice := adviseflags[uint(args[2])]

	callrep := fmt.Sprintf("%d madvise(0x%X, %d, %s)", args[2], addr, size, advice)
	return callrep, nil

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

func render_munmap(pid int, args RegisterArgs) (string, error) {

	addr := args[0]
	size := args[1]

	callrep := fmt.Sprintf("munmap(0x%X, %d)", addr, size)

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
