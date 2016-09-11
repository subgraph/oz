package native

import (
	"syscall"
	"unsafe"

	"github.com/twtiger/gosecco/constants"
	"github.com/twtiger/gosecco/data"
)

// #include <linux/seccomp.h>
// #include <sys/prctl.h>
import "C"

// seccomp is a wrapper for the 'seccomp' system call.
// See <linux/seccomp.h> for valid op and flag values.
// uargs is typically a pointer to struct sock_fprog.
func seccomp(op, flags uintptr, uargs unsafe.Pointer) error {
	nr, _ := constants.GetSyscall("seccomp")
	_, _, e := syscall.Syscall(uintptr(nr), op, flags, uintptr(uargs))
	if e != 0 {
		return e
	}
	return nil
}

// InstallSeccomp will install seccomp using native methods
func InstallSeccomp(prog *data.SockFprog) error {
	return seccomp(C.SECCOMP_SET_MODE_FILTER, C.SECCOMP_FILTER_FLAG_TSYNC, unsafe.Pointer(prog))
}

// prctl is a wrapper for the 'prctl' system call.
// See 'man prctl' for details.
func prctl(option uintptr, args ...uintptr) error {
	if len(args) > 4 {
		return syscall.E2BIG
	}
	var arg [4]uintptr
	copy(arg[:], args)
	_, _, e := syscall.Syscall6(syscall.SYS_PRCTL, option, arg[0], arg[1], arg[2], arg[3], 0)
	if e != 0 {
		return e
	}
	return nil
}

// NoNewPrivs will use prctl to stop new privileges using native methods
func NoNewPrivs() error {
	return prctl(C.PR_SET_NO_NEW_PRIVS, 1)
}

// CheckGetSeccomp will check if we have seccomp available
func CheckGetSeccomp() error {
	return prctl(syscall.PR_GET_SECCOMP)
}

// CheckSetSeccompModeFilter will check if we have seccomp mode filter available
func CheckSetSeccompModeFilter() error {
	return prctl(syscall.PR_SET_SECCOMP, C.SECCOMP_MODE_FILTER, 0)
}

// CheckSetSeccompModeFilterWithSeccomp will check if we have the seccomp syscall available
func CheckSetSeccompModeFilterWithSeccomp() error {
	return seccomp(C.SECCOMP_SET_MODE_FILTER, 0, nil)
}

// CheckSetSeccompModeTsync will check that we can set tsync
func CheckSetSeccompModeTsync() error {
	return seccomp(C.SECCOMP_SET_MODE_FILTER, C.SECCOMP_FILTER_FLAG_TSYNC, nil)
}
