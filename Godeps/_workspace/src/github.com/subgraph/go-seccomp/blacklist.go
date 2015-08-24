// Extending for Blacklist support in Subgraph Oz
// Much refactoring on this remains to be done.
// Original copyright and license below.
//
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
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// #include <sys/prctl.h>
// #include "unistd_64.h"
// #include "seccomp.h"
import "C"

// parseLine parses the policy line for a single syscall.
func parseLineBlacklist(line string, enforce bool) (policy, error) {
	var name, expr, ret string
	var then SockFilter
	var def *SockFilter

	line = strings.TrimSpace(line)
	if match := allowRE.FindStringSubmatch(line); match != nil {
		name = match[1]
		if enforce == true {
			def = ptr(bpfRet(retKill()))
		} else {
			def = ptr(bpfRet(retTrace()))
		}
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

	if enforce == true {
		then = bpfRet(retKill())
	} else {
		then = bpfRet(retTrace())
	}

	if ret != "" {
		errno, err := strconv.ParseUint(ret, 0, 16)
		if err != nil {
			return policy{}, fmt.Errorf("invalid errno: %s", ret)
		}
		def = ptr(bpfRet(retErrno(syscall.Errno(errno))))
	}

	return policy{name, or, then, def}, nil
}

// parseLines parses multiple policy lines, each one for a single syscall.
// Empty lines and lines beginning with "#" are ignored.
// Multiple policies for a syscall are detected and reported as error.
func parseLinesBlacklist(lines []string, enforce bool) ([]policy, error) {
	var ps []policy
	seen := make(map[string]int)
	for i, line := range lines {
		lineno := i + 1
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := parseLineBlacklist(line, enforce)
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
func parseFileBlacklist(path string, enforce bool) ([]policy, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseLinesBlacklist(strings.Split(string(file), "\n"), enforce)
}

// compile compiles a Seccomp-BPF program implementing the syscall policies.
// long specifies whether to generate 32-bit or 64-bit argument comparisons.
// def is the overall default action to take when the syscall does not match
// any policy in the filter.
func compileBlacklist(ps []policy, long bool, def SockFilter, enforce bool) ([]SockFilter, error) {
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
func CompileBlacklist(path string, enforce bool) ([]SockFilter, error) {
	ps, err := parseFileBlacklist(path, enforce)
	if err != nil {
		return nil, err
	}
	return compileBlacklist(ps, nbits > 32, bpfRet(retAllow()), enforce)
}

// Install makes the necessary system calls to install the Seccomp-BPF
// filter for the current process (all threads). Install can be called
// multiple times to install additional filters.
func InstallBlacklist(bpf []SockFilter) error {
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
