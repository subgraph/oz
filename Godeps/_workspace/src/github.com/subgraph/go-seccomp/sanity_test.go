// Copyright 2015 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package seccomp

import (
	"testing"
	"unsafe"
)

func TestSockFilter(t *testing.T) {
	var f SockFilter
	var cf sock_filter
	tests := []struct {
		desc string
		g, c uintptr
	}{
		{
			"Sizeof(SockFilter)",
			unsafe.Sizeof(f),
			unsafe.Sizeof(cf),
		},
		{
			"Sizeof(SockFilter.Code)",
			unsafe.Sizeof(f.Code),
			unsafe.Sizeof(cf.code),
		},
		{
			"Offsetof(SockFilter.Code)",
			unsafe.Offsetof(f.Code),
			unsafe.Offsetof(cf.code),
		},
		{
			"Sizeof(SockFilter.Jt)",
			unsafe.Sizeof(f.JT),
			unsafe.Sizeof(cf.jt),
		},
		{
			"Offsetof(SockFilter.Jt)",
			unsafe.Offsetof(f.JT),
			unsafe.Offsetof(cf.jt),
		},
		{
			"Sizeof(SockFilter.Jf)",
			unsafe.Sizeof(f.JF),
			unsafe.Sizeof(cf.jf),
		},
		{
			"Offsetof(SockFilter.Jf)",
			unsafe.Offsetof(f.JF),
			unsafe.Offsetof(cf.jf),
		},
		{
			"Sizeof(SockFilter.K)",
			unsafe.Sizeof(f.K),
			unsafe.Sizeof(cf.k),
		},
		{
			"Offsetof(SockFilter.K)",
			unsafe.Offsetof(f.K),
			unsafe.Offsetof(cf.k),
		},
	}
	for _, test := range tests {
		if test.g != test.c {
			t.Errorf("%s = %v; want %v", test.desc, test.g, test.c)
		}
	}
}

func TestSockFprog(t *testing.T) {
	var p SockFprog
	var cp sock_fprog
	tests := []struct {
		desc string
		g, c uintptr
	}{
		{
			"Sizeof(SockFprog)",
			unsafe.Sizeof(p),
			unsafe.Sizeof(cp),
		},
		{
			"Sizeof(SockFprog.Len)",
			unsafe.Sizeof(p.Len),
			unsafe.Sizeof(cp.len),
		},
		{
			"Offsetof(SockFprog.Len)",
			unsafe.Offsetof(p.Len),
			unsafe.Offsetof(cp.len),
		},
		{
			"Sizeof(SockFprog.Filter)",
			unsafe.Sizeof(p.Filter),
			unsafe.Sizeof(cp.filter),
		},
		{
			"Offsetof(SockFprog.Filter)",
			unsafe.Offsetof(p.Filter),
			unsafe.Offsetof(cp.filter),
		},
	}
	for _, test := range tests {
		if test.g != test.c {
			t.Errorf("%s = %v; want %v", test.desc, test.g, test.c)
		}
	}
}

func TestSeccompData(t *testing.T) {
	var d SeccompData
	var cd seccomp_data
	tests := []struct {
		desc string
		g, c uintptr
	}{
		{
			"Sizeof(SeccompData)",
			unsafe.Sizeof(d),
			unsafe.Sizeof(cd),
		},
		{
			"Sizeof(SeccompData.NR)",
			unsafe.Sizeof(d.NR),
			unsafe.Sizeof(cd.nr),
		},
		{
			"Offsetof(SeccompData.NR)",
			unsafe.Offsetof(d.NR),
			unsafe.Offsetof(cd.nr),
		},
		{
			"Sizeof(SeccompData.Arch)",
			unsafe.Sizeof(d.Arch),
			unsafe.Sizeof(cd.arch),
		},
		{
			"Offsetof(SeccompData.Arch)",
			unsafe.Offsetof(d.Arch),
			unsafe.Offsetof(cd.arch),
		},
		{
			"Sizeof(SeccompData.InstructionPointer)",
			unsafe.Sizeof(d.InstructionPointer),
			unsafe.Sizeof(cd.instruction_pointer),
		},
		{
			"Offsetof(SeccompData.InstructionPointer)",
			unsafe.Offsetof(d.InstructionPointer),
			unsafe.Offsetof(cd.instruction_pointer),
		},
		{
			"Sizeof(SeccompData.Args)",
			unsafe.Sizeof(d.Args),
			unsafe.Sizeof(cd.args),
		},
		{
			"Offsetof(SeccompData.Args)",
			unsafe.Offsetof(d.Args),
			unsafe.Offsetof(cd.args),
		},
		{
			"Sizeof(SeccompData.Args[0])",
			unsafe.Sizeof(d.Args[0]),
			unsafe.Sizeof(cd.args[0]),
		},
	}
	for _, test := range tests {
		if test.g != test.c {
			t.Errorf("%s = %v; want %v", test.desc, test.g, test.c)
		}
	}
}
