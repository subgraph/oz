// Copyright 2015 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package seccomp

import (
	"fmt"
)

// #include <linux/filter.h>
import "C"

// BPF machine opcodes. These are the only ones we use.
const (
	opLOAD = C.BPF_LD + C.BPF_W + C.BPF_ABS
	opJGT = C.BPF_JMP + C.BPF_JGT + C.BPF_K
	opJEQ  = C.BPF_JMP + C.BPF_JEQ + C.BPF_K
	opJSET = C.BPF_JMP + C.BPF_JSET + C.BPF_K
	opJUMP = C.BPF_JMP + C.BPF_JA
	opRET  = C.BPF_RET + C.BPF_K
)

// SockFilter encodes one BPF machine instruction.
// This struct mirrors struct sock_filter from <linux/filter.h>.
type SockFilter struct {
	Code uint16 // Actual filter code.
	JT   uint8  // Jump true.
	JF   uint8  // Jump false.
	K    uint32 // Generic multiuse field.
}

// SockFprog encodes a BPF machine program.
// This struct mirrors struct sock_fprog from <linux/filter.h>.
type SockFprog struct {
	Len    uint16      // Number of BPF machine instructions.
	Filter *SockFilter // Pointer to the first instruction.
}

// C versions of the structs used for sanity checking.
type sock_filter C.struct_sock_filter
type sock_fprog C.struct_sock_fprog

// bpfInsn constructs one BPF machine instruction.
func bpfInsn(code uint16, k uint32, jt, jf uint8) SockFilter {
	return SockFilter{code, jt, jf, k}
}

// bpfStmt constructs one BPF machine statement.
func bpfStmt(code uint16, k uint32) SockFilter {
	return bpfInsn(code, k, 0, 0)
}

// bpfLoad returns the instruction to load the word at the given offset.
func bpfLoad(offset uintptr) SockFilter {
	return bpfStmt(opLOAD, uint32(offset))
}

// bpfJeq returns an instruction encoding "jump-if-equal".
// Register A is compared with val.
// Both jt and jf are relative offsets. Offset 0 means fallthrough.
func bpfJeq(val uint32, jt, jf uint8) SockFilter {
	return bpfInsn(opJEQ, val, jt, jf)
}

// bpfJgt returns an instruction encoding "jump-if-greater-than".
func bpfJgt(val uint32, jt, jf uint8) SockFilter {
	return bpfInsn(opJGT, val, jt, jf)
}

// bpfJset returns an instruction encoding "jump-if-set".
// Register A is bitwise anded with val and result compared with zero.
// Both jt and jf are relative offsets. Offset 0 means fallthrough.
func bpfJset(val uint32, jt, jf uint8) SockFilter {
	return bpfInsn(opJSET, val, jt, jf)
}

// bpfJump returns an instruction encoding an unconditional jump to a relative offset.
// Offset 0 means fallthrough (NOP).
func bpfJump(offset int) SockFilter {
	return bpfStmt(opJUMP, uint32(offset))
}

// bpfRet returns the instruction to return the value val.
func bpfRet(val uint32) SockFilter {
	return bpfStmt(opRET, val)
}

// String returns a readable representation of a BPF machine instruction.
func (f SockFilter) String() string {
	var code string
	switch f.Code {
	case opLOAD:
		code = "Load"
	case opJEQ:
		code = "Jeq"
	case opJSET:
		code = "Jset"
	case opJGT:
		code = "Jgt"
	case opJUMP:
		code = "Jump"
	case opRET:
		code = "Return"
	default:
		code = fmt.Sprintf("%04x", f.Code)
	}
	return fmt.Sprintf("%8s %08x, %02x, %02x\n", code, f.K, f.JT, f.JF)
}

// ptr returns a pointer to a copy of the argument, useful in cases
// where the & syntax isn't allowed. e.g. ptr(bpfInsn(...)).
func ptr(f SockFilter) *SockFilter {
	return &f
}
