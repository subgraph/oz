// Copyright 2015 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package seccomp implements support for compiling and installing Seccomp-BPF policy files.
//   - http://www.chromium.org/chromium-os/developer-guide/chromium-os-sandboxing
//
// Typical usage:
//	// Check for the required kernel support for seccomp.
//	if err := seccomp.CheckSupport(); err != nil {
//		log.Fatal(err)
//	}
//
//	// Compile BPF program from a Chromium-OS policy file.
//	bpf, err := seccomp.Compile(path)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Install Seccomp-BPF filter program with the kernel.
//	if err := seccomp.Install(bpf); err != nil {
//		log.Fatal(err)
//	}
//
// For background and more information:
//   - http://www.tcpdump.org/papers/bpf-usenix93.pdf
//   - http://en.wikipedia.org/wiki/Seccomp
//   - http://lwn.net/Articles/475043/
//   - http://outflux.net/teach-seccomp/
//   - http://www.kernel.org/doc/Documentation/prctl/seccomp_filter.txt
//   - http://github.com/torvalds/linux/blob/master/kernel/seccomp.c
//
// TODO:
//   - Exit the program if any thread is killed because of seccomp violation.
//   - Provide a debug mode to log system calls used during normal operation.
package seccomp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// #include <sys/prctl.h>
// #include "unistd_64.h"
// #include "seccomp.h"
import "C"

// SeccompData is the format the BPF program executes over.
// This struct mirrors struct seccomp_data from <linux/seccomp.h>.
type SeccompData struct {
	NR                 int32     // The system call number.
	Arch               uint32    // System call convention as an AUDIT_ARCH_* value.
	InstructionPointer uint64    // At the time of the system call.
	Args               [6]uint64 // System call arguments (always stored as 64-bit values).
}

// C version of the struct used for sanity checking.
type seccomp_data C.struct_seccomp_data

// bpfLoadNR returns the instruction to load the NR field in SeccompData.
func bpfLoadNR() SockFilter {
	return bpfLoad(unsafe.Offsetof(SeccompData{}.NR))
}

// bpfLoadArch returns the instruction to load the Arch field in SeccompData.
func bpfLoadArch() SockFilter {
	return bpfLoad(unsafe.Offsetof(SeccompData{}.Arch))
}

// bpfLoadArg returns the instruction to load one word of an argument in SeccompData.
func bpfLoadArg(arg, word int) SockFilter {
	return bpfLoad(unsafe.Offsetof(SeccompData{}.Args) + uintptr(((2*arg)+word)*4))
}

// retKill returns the code for seccomp kill action.
func retKill() uint32 {
	return C.SECCOMP_RET_KILL
}

// retTrap returns the code for seccomp trap action.
func retTrap() uint32 {
	return C.SECCOMP_RET_TRAP
}

// retErrno returns the code for seccomp errno action with the specified errno embedded.
func retErrno(errno syscall.Errno) uint32 {
	return C.SECCOMP_RET_ERRNO | (uint32(errno) & C.SECCOMP_RET_DATA)
}

// retAllow returns the code for seccomp allow action.
func retAllow() uint32 {
	return C.SECCOMP_RET_ALLOW
}

func retTrace() uint32 {
	return C.SECCOMP_RET_TRACE
}
// policy represents the seccomp policy for a single syscall.
type policy struct {
	// name of the syscall.
	name string

	// expr is evaluated on the syscall arguments.
	// nil expr evaluates to false.
	expr orExpr

	// then is executed if the expr evaluates to true.
	// (cannot be specified in policy file, used in tests only).
	then SockFilter

	// default action (else) if the expr evaluates to false.
	// nil means jump to end of program for the overall default.
	def *SockFilter
}

// orExpr is a list of and expressions.
type orExpr []andExpr

// andExpr is a list of arg comparisons.
type andExpr []argComp

// argComp represents a basic argument comparison in the policy.
type argComp struct {
	idx  int    // 0..5 for indexing into SeccompData.Args.
	oper string // comparison operator: "==", "!=", or "&".
	val  uint64 // upper 32 bits compared only if nbits>32.
}

// String converts the internal policy representation back to policy file syntax.
func (p policy) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s: ", p.name)

	for i, and := range p.expr {
		if i > 0 {
			fmt.Fprintf(&buf, " || ")
		}
		for j, arg := range and {
			if j > 0 {
				fmt.Fprintf(&buf, " && ")
			}
			fmt.Fprintf(&buf, "arg%d %s %#x", arg.idx, arg.oper, arg.val)
		}
	}

	pret := func(f SockFilter) {
		if f.Code == opRET {
			switch f.K & C.SECCOMP_RET_ACTION {
			case C.SECCOMP_RET_ALLOW:
				fmt.Fprintf(&buf, "1")
				return
			case C.SECCOMP_RET_ERRNO:
				fmt.Fprintf(&buf, "return %d", f.K&C.SECCOMP_RET_DATA)
				return
			}
		}
		fmt.Fprintf(&buf, "%s", f)
	}
	if p.then != bpfRet(retAllow()) {
		fmt.Fprintf(&buf, " ? ")
		pret(p.then)
	}
	if p.def != nil {
		if p.expr != nil {
			fmt.Fprintf(&buf, "; ")
		}
		pret(*p.def)
	}

	return buf.String()
}

// Syntax of policy line for a single syscall.
var (
	allowRE      = regexp.MustCompile(`^([[:word:]]+) *: *1$`)
	returnRE     = regexp.MustCompile(`^([[:word:]]+) *: *return *([[:word:]]+)$`)
	exprRE       = regexp.MustCompile(`^([[:word:]]+) *:([^;]+)$`)
	exprReturnRE = regexp.MustCompile(`^([[:word:]]+) *:([^;]+); *return *([[:word:]]+)$`)

	argRE = regexp.MustCompile(`^arg([0-5]) *(==|!=|&) *([[:word:]]+)$`)
)

// parseLine parses the policy line for a single syscall.
func parseLine(line string, enforce bool) (policy, error) {
	var name, expr, ret string
	var then SockFilter
	var def *SockFilter

	line = strings.TrimSpace(line)
	if match := allowRE.FindStringSubmatch(line); match != nil {
		name = match[1]
		def = ptr(bpfRet(retAllow()))
	} else if match = returnRE.FindStringSubmatch(line); match != nil {
		name = match[1]
		ret = match[2]
	} else if match = exprRE.FindStringSubmatch(line); match != nil {
		name = match[1]
		expr = match[2]
	} else if match = exprReturnRE.FindStringSubmatch(line); match != nil {
		name = match[1]
		expr = match[2]
		ret = match[3]
	} else {
		return policy{}, fmt.Errorf("invalid syntax")
	}

	if _, ok := syscallNum[name]; !ok {
		return policy{}, fmt.Errorf("unknown syscall: %s", name)
	}

	var or orExpr
	if expr != "" {
		for _, sub := range strings.Split(expr, "||") {
			var and andExpr
			for _, arg := range strings.Split(sub, "&&") {
				arg = strings.TrimSpace(arg)
				match := argRE.FindStringSubmatch(arg)
				if match == nil {
					return policy{}, fmt.Errorf("invalid expression: %s", arg)
				}
				idx, err := strconv.Atoi(match[1])
				if err != nil {
					return policy{}, fmt.Errorf("invalid arg: %s", arg)
				}
				oper := match[2]
				val, err := strconv.ParseUint(match[3], 0, 64)
				if err != nil {
					return policy{}, fmt.Errorf("invalid value: %s", arg)
				}
				and = append(and, argComp{idx, oper, val})
			}
			or = append(or, and)
		}
	}
	then = bpfRet(retAllow())
	if ret != "" {
		errno, err := strconv.ParseUint(ret, 0, 16)
		if err != nil {
			return policy{}, fmt.Errorf("invalid errno: %s", ret)
		}
		if (enforce == false) {
			def = ptr(bpfRet(retTrace()))
		} else {
			def = ptr(bpfRet(retErrno(syscall.Errno(errno))))
		}
	}

	return policy{name, or, then, def}, nil
}

// parseLines parses multiple policy lines, each one for a single syscall.
// Empty lines and lines beginning with "#" are ignored.
// Multiple policies for a syscall are detected and reported as error.
func parseLines(lines []string, enforce bool) ([]policy, error) {
	var ps []policy
	seen := make(map[string]int)
	for i, line := range lines {
		lineno := i + 1
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := parseLine(line, enforce)
		if err != nil {
			return nil, fmt.Errorf("line %d: %v", lineno, err)
		}
		if seen[p.name] > 0 {
			return nil, fmt.Errorf("lines %d,%d: multiple policies for %s",
				seen[p.name], lineno, p.name)
		}
		seen[p.name] = lineno
		ps = append(ps, p)
	}
	return ps, nil
}

// parseFile reads a Chromium-OS Seccomp-BPF policy file and parses its contents.
func parseFile(path string, enforce bool) ([]policy, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseLines(strings.Split(string(file), "\n"), enforce)
}

// compile compiles a Seccomp-BPF program implementing the syscall policies.
// long specifies whether to generate 32-bit or 64-bit argument comparisons.
// def is the overall default action to take when the syscall does not match
// any policy in the filter.
func compile(ps []policy, long bool, def SockFilter, enforce bool) ([]SockFilter, error) {
	var bpf []SockFilter
	do := func(insn SockFilter) {
		bpf = append(bpf, insn)
	}

	// ref maps a label to addresses of all the instructions that jump to it.
	ref := make(map[string][]int)
	jump := func(name string) {
		// jump to a label with unresolved address: insert a placeholder instruction.
		ref[name] = append(ref[name], len(bpf))
		do(SockFilter{})
	}
	label := func(name string) {
		// label address resolved: replace placeholder instructions with actual jumps.
		for _, i := range ref[name] {
			bpf[i] = bpfJump(len(bpf) - (i + 1))
		}
		delete(ref, name)
	}

	// Conditional jumps: jump if condition is true, fall through otherwise.
	jeq := func(val uint32, target string) {
		// if A == val { goto target }
		do(bpfJeq(val, 0, 1))
		jump(target)
	}
	jne := func(val uint32, target string) {
		// if A != val { goto target }
		do(bpfJeq(val, 1, 0))
		jump(target)
	}
	jset := func(val uint32, target string) {
		// if A&val != 0 { goto target }
		do(bpfJset(val, 0, 1))
		jump(target)
	}
	jnset := func(val uint32, target string) {
		// if A&val == 0 { goto target }
		do(bpfJset(val, 1, 0))
		jump(target)
	}

	do(bpfLoadArch())
	do(bpfJeq(auditArch, 1, 0))
	do(bpfRet(retKill()))

	do(bpfLoadNR())
	for _, p := range ps {
		nr, ok := syscallNum[p.name]
		if !ok {
			return nil, fmt.Errorf("unknown syscall: %s", p.name)
		}
		jne(uint32(nr), "nextcall")
		for _, and := range p.expr {
			for _, arg := range and {
				val := struct{ high, low uint32 }{uint32(arg.val >> 32), uint32(arg.val)}
				switch arg.oper {
				case "==":
					if long {
						do(bpfLoadArg(arg.idx, 1))
						jne(val.high, "nextor")
					}
					do(bpfLoadArg(arg.idx, 0))
					jne(val.low, "nextor")
				case "!=":
					if long {
						do(bpfLoadArg(arg.idx, 1))
						jne(val.high, "nextand")
					}
					do(bpfLoadArg(arg.idx, 0))
					jeq(val.low, "nextor")
				case "&":
					if long {
						do(bpfLoadArg(arg.idx, 1))
						jset(val.high, "nextand")
					}
					do(bpfLoadArg(arg.idx, 0))
					jnset(val.low, "nextor")
				default:
					return nil, fmt.Errorf("unknown operator: %q", arg.oper)
				}

				// Comparison was satisfied. Move on to the next comparison in &&.
				label("nextand")
			}

			// All comparisons in && were satisfied.
			do(p.then)

			// Some comparison in && was false. Move on to the next expression in ||.
			label("nextor")
		}

		// All expressions in || evaluated to false (or expr was nil).
		if p.def != nil {
			do(*p.def)
		} else {
			jump("default")
		}

		label("nextcall")
	}

	label("default")
	do(def)

	if len(ref) > 0 {
		return nil, fmt.Errorf("unresolved labels: %v\n%v", ref, bpf)
	}
	return bpf, nil
}

// Compile reads a Chromium-OS policy file and compiles a
// Seccomp-BPF filter program implementing the policies.
func Compile(path string, enforce bool) ([]SockFilter, error) {
	ps, err := parseFile(path, enforce)
	if err != nil {
		return nil, err
	}
	var op SockFilter
	if enforce == true {
		op = bpfRet(retKill())
	} else {
		op = bpfRet(retTrace())
	}
	return compile(ps, nbits > 32, op, enforce)
}

// prctl is a wrapper for the 'prctl' system call.
// See 'man prctl' for details.
func prctl(option uintptr, args ...uintptr) error {
	if len(args) > 4 {
		return syscall.E2BIG
	}
	var arg [4]uintptr
	copy(arg[:], args)
	_, _, e := syscall.Syscall6(C.__NR_prctl, option, arg[0], arg[1], arg[2], arg[3], 0)
	if e != 0 {
		return e
	}
	return nil
}

// seccomp is a wrapper for the 'seccomp' system call.
// See <linux/seccomp.h> for valid op and flag values.
// uargs is typically a pointer to struct sock_fprog.
func seccomp(op, flags uintptr, uargs unsafe.Pointer) error {
	_, _, e := syscall.Syscall(C.__NR_seccomp, op, flags, uintptr(uargs))
	if e != 0 {
		return e
	}
	return nil
}

// CheckSupport checks for the required seccomp support in the kernel.
func CheckSupport() error {
	// This is based on http://outflux.net/teach-seccomp/autodetect.html.
	if err := prctl(C.PR_GET_SECCOMP); err != nil {
		return fmt.Errorf("seccomp not available: %v", err)
	}
	if err := prctl(C.PR_SET_SECCOMP, C.SECCOMP_MODE_FILTER, 0); err != syscall.EFAULT {
		return fmt.Errorf("seccomp filter not available: %v", err)
	}
	if err := seccomp(C.SECCOMP_SET_MODE_FILTER, 0, nil); err != syscall.EFAULT {
		return fmt.Errorf("seccomp syscall not available: %v", err)
	}
	if err := seccomp(C.SECCOMP_SET_MODE_FILTER, C.SECCOMP_FILTER_FLAG_TSYNC, nil); err != syscall.EFAULT {
		return fmt.Errorf("seccomp tsync not available: %v", err)
	}
	return nil
}

// Load makes the seccomp system call to install the bpf filter for
// all threads (with tsync). prctl(set_no_new_privs, 1) must have
// been called (from the same thread) before calling Load for the
// first time.
//   Most users of this library should use Install instead of calling
//   Load directly. There are a couple of situations where it may be
//   necessary to use Load instead of Install:
//   - If a previous call to Install has disabled the 'prctl' system
//     call, Install cannot be called again. In that case, it is safe
//     to add additional filters directly with Load.
//   - If the process is running as a priviledged user, and you want
//     to load the seccomp filter without setting no_new_privs.
func Load(bpf []SockFilter) error {
	if size, limit := len(bpf), 0xffff; size > limit {
		return fmt.Errorf("filter program too big: %d bpf instructions (limit = %d)", size, limit)
	}
	prog := &SockFprog{
		Filter: &bpf[0],
		Len:    uint16(len(bpf)),
	}
	return seccomp(C.SECCOMP_SET_MODE_FILTER, C.SECCOMP_FILTER_FLAG_TSYNC, unsafe.Pointer(prog))
}

// Install makes the necessary system calls to install the Seccomp-BPF
// filter for the current process (all threads). Install can be called
// multiple times to install additional filters.
func Install(bpf []SockFilter) error {
	// prctl(set_no_new_privs, 1) must be called (from the same thread)
	// before a seccomp filter can be installed by an unprivileged user:
	// - http://www.kernel.org/doc/Documentation/prctl/no_new_privs.txt.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := prctl(C.PR_SET_NO_NEW_PRIVS, 1); err != nil {
		return err
	}
	return Load(bpf)
}
