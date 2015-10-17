package seccomp

import (
	"syscall"
)

type RenderingFunctions map[int]func(int, RegisterArgs) (string, error)

func getRenderingFunctions() RenderingFunctions {
	r := map[int]func(pid int, args RegisterArgs) (string, error){
		syscall.SYS_ACCESS:   render_access,
		syscall.SYS_MPROTECT: render_mprotect,
		syscall.SYS_MMAP:     render_mmap,
		syscall.SYS_MREMAP:   render_mremap,
		syscall.SYS_MUNMAP:   render_munmap,
		syscall.SYS_FUTEX:    render_futex,
		syscall.SYS_OPENAT:   render_openat,
		syscall.SYS_OPEN:     render_open,
		syscall.SYS_MADVISE:  render_madvise,
	}
	return r
}

func renderFlags(flags map[uint]string, val uint) string {
	found := false
	flagstr := ""

	for flag := range flags {
		if val&uint(flag) == uint(flag) {
			if found == true {
				flagstr += "|"
			}
			flagstr += flags[flag]
			found = true
		}
	}
	return flagstr

}

func allFlagsTest(flags []uint, val uint) bool {
	var i uint = 0

	for flag := range flags {
		i |= uint(flag)
	}
	return i == val
}
