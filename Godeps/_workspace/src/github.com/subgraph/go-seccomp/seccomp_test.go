// Copyright 2015 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package seccomp

import (
	"flag"
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

var (
	parseFilePath = flag.String("parse_file", "", "path for TestParseFile")
	testInstall   = flag.Bool("test_install", false, "enable TestInstall (use with -cpu=N)")
	testEndian    = flag.Bool("test_endian", false, "enable TestEndian")
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		policy string
		want   string
		err    string
	}{
		{policy: "read: 1"},
		{policy: "open: return 1"},
		{policy: "prctl: arg0 == 0xf"},
		{policy: "prctl: arg0 != 0xf"},
		{policy: "ioctl: arg1 & 0x5401"},
		{policy: "ioctl: arg1 == 0x4024700a || arg1 == 0x541b"},
		{policy: "ioctl: arg1 == 0x4024700a && arg1 == 0x541b"},
		{policy: "ioctl: arg1 == 0x5401 || arg1 == 0x700a || arg2 & 0x541b"},
		{policy: "ioctl: arg1 == 0x5401 && arg1 == 0x700a && arg3 & 0x541b"},
		{policy: "ioctl: arg1 == 0x5401 || arg1 == 0x700a && arg4 & 0x541b"},
		{policy: "ioctl: arg1 == 0x5401 && arg1 == 0x700a || arg5 & 0x541b"},
		{policy: "ioctl: arg1 == 0x5401 && arg1 == 0x700a || arg5 & 0x541b; return 1"},
		{
			// different spacing around colon.
			policy: "read :1",
			want:   "read: 1",
		},
		{
			// leading and trailing whitespace.
			policy: " open :  return 1  ",
			want:   "open: return 1",
		},
		{
			// return hexadecimal errno.
			policy: "open: return 0x10",
			want:   "open: return 16",
		},
		{
			// return octal errno.
			policy: "open: return 010",
			want:   "open: return 8",
		},
		{
			// return highest errno.
			policy: "open: return 0xffff",
			want:   "open: return 65535",
		},
		{
			// expression with no spaces.
			policy: "ioctl:arg1==0x5401&&arg1==0x700a||arg2&0x541b",
			want:   "ioctl: arg1 == 0x5401 && arg1 == 0x700a || arg2 & 0x541b",
		},
		{
			// compare with decimal value.
			policy: "ioctl: arg1 == 5401 && arg1 == 0x700a || arg2 & 0x541b",
			want:   "ioctl: arg1 == 0x1519 && arg1 == 0x700a || arg2 & 0x541b",
		},
		{
			// compare with octal value.
			policy: "ioctl: arg1 == 05401 && arg1 == 0x700a || arg2 & 0x541b",
			want:   "ioctl: arg1 == 0xb01 && arg1 == 0x700a || arg2 & 0x541b",
		},
		{
			// all decimal comparisons.
			policy: "clone: arg0 == 1 || arg0 == 2 || arg0 == 16",
			want:   "clone: arg0 == 0x1 || arg0 == 0x2 || arg0 == 0x10",
		},
		{
			// missing syscall name.
			policy: ": 1",
			err:    "invalid syntax",
		},
		{
			// malformed syscall name.
			policy: "two words: 1",
			err:    "invalid syntax",
		},
		{
			// missing colon.
			policy: "read = 1",
			err:    "invalid syntax",
		},
		{
			// trailing semicolon after return.
			policy: "open: return 1;",
			err:    "invalid syntax",
		},
		{
			// trailing semicolon after expression.
			policy: "prctl: arg0 == 0xf;",
			err:    "invalid syntax",
		},
		{
			// missing return after semicolon.
			policy: "prctl: arg0 == 0xf; 1",
			err:    "invalid syntax",
		},
		{
			// bad syscall name.
			policy: "bad: 1",
			err:    "unknown syscall: bad",
		},
		{
			// symbolic errno is not supported.
			policy: "open: return EPERM",
			err:    "invalid errno: EPERM",
		},
		{
			// errno must fit in 16 bits.
			policy: "open: return 0x10000",
			err:    "invalid errno: 0x10000",
		},
		{
			// missing argument index.
			policy: "prctl: arg == 0xf",
			err:    "invalid expression: arg == 0xf",
		},
		{
			// arg index out of range.
			policy: "prctl: arg6 == 0xf",
			err:    "invalid expression: arg6 == 0xf",
		},
		{
			// bitwise and with argument not supported.
			policy: "prctl: arg0 & 0xf == 0xf",
			err:    "invalid expression: arg0 & 0xf == 0xf",
		},
		{
			// unknown operator.
			policy: "prctl: arg0 !== 0xf",
			err:    "invalid expression: arg0 !== 0xf",
		},
		{
			// invalid hexadecimal value.
			policy: "prctl: arg0 == 0xfdx",
			err:    "invalid value: arg0 == 0xfdx",
		},
		{
			// invalid decimal value.
			policy: "prctl: arg0 == 123a",
			err:    "invalid value: arg0 == 123a",
		},
		{
			// invalid octal value.
			policy: "prctl: arg0 == 0129",
			err:    "invalid value: arg0 == 0129",
		},
		{
			// invalid subexpression.
			policy: "prctl: arg0 == 0x100 && arg1 = 0x101 || arg2 == 0x102",
			err:    "invalid expression: arg1 = 0x101",
		},
	}
	for _, test := range tests {
		var err string
		p, e := parseLine(test.policy)
		if e != nil {
			err = e.Error()
		}
		if err != "" || test.err != "" {
			if err != test.err {
				t.Errorf("parseLine(%q): error = %q; want %q", test.policy, err, test.err)
			}
			continue
		}
		want := test.want
		if want == "" {
			want = test.policy
		}
		if got := p.String(); got != want {
			t.Errorf("parseLine(%q) = %q; want %q", test.policy, got, test.want)
		}
	}
}

func TestParseLines(t *testing.T) {
	tests := []struct {
		file []string
		err  string
	}{
		{
			// simple policy file.
			file: []string{
				"read: 1",
				"write: 1",
				"open: return 1",
			},
		},
		{
			// comment lines are ignored.
			file: []string{
				"read: 1",
				"write: 1",
				"# open: return EPERM",
				"open: return 1",
			},
		},
		{
			// blank lines are ignored.
			file: []string{
				"read: 1",
				"write: 1",
				"",
				"open: return 1",
			},
		},
		{
			// leading space on comment line.
			file: []string{
				"read: 1",
				"write: 1",
				" # open: return EPERM",
				"open: return 1",
			},
			err: "line 3: invalid syntax",
		},
		{
			// line consisting of whitespace only.
			file: []string{
				"read: 1",
				"write: 1",
				" ",
				"open: return 1",
			},
			err: "line 3: invalid syntax",
		},
		{
			// parse error on one line.
			file: []string{
				"read: 1",
				"write: return 019",
				"open: return 1",
			},
			err: "line 2: invalid errno: 019",
		},
		{
			// multiple policies for a syscall.
			file: []string{
				"read: 1",
				"write: 1",
				"read: return 1",
				"open: return 1",
			},
			err: "lines 1,3: multiple policies for read",
		},
	}
	for _, test := range tests {
		var err string
		_, e := parseLines(test.file)
		if e != nil {
			err = e.Error()
		}
		if err != "" || test.err != "" {
			if err != test.err {
				t.Errorf("parseLines(%q): error = %q; want %q", test.file, err, test.err)
			}
		}
	}
}

func TestParseFile(t *testing.T) {
	if *parseFilePath == "" {
		t.Skip("use -parse_file to enable.")
	}

	if _, err := parseFile(*parseFilePath); err != nil {
		t.Errorf("parseFile(%q): %v", *parseFilePath, err)
	}
}

func TestCompile(t *testing.T) {
	syscallName := make(map[int32]string)
	for name, nr := range syscallNum {
		syscallName[int32(nr)] = name
	}

	call := func(name string, args ...uint64) SeccompData {
		nr, ok := syscallNum[name]
		if !ok {
			t.Fatalf("unknown syscall: %s", name)
		}
		data := SeccompData{
			NR:   int32(nr),
			Arch: auditArch,
		}
		copy(data.Args[:], args)
		return data
	}

	eval := func(bpf []SockFilter, data SeccompData) uint32 {
		var A uint32
		IP := 0
		for {
			Insn := bpf[IP]
			IP++
			switch Insn.Code {
			case opLOAD:
				A = *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&data)) + uintptr(Insn.K)))
			case opJEQ:
				if A == Insn.K {
					IP += int(Insn.JT)
				} else {
					IP += int(Insn.JF)
				}
			case opJSET:
				if A&Insn.K != 0 {
					IP += int(Insn.JT)
				} else {
					IP += int(Insn.JF)
				}
			case opJUMP:
				IP += int(Insn.K)
			case opRET:
				return Insn.K
			default:
				t.Fatalf("unsupported instruction: %v", Insn)
			}
		}
	}

	file := []string{
		"read: 1",
		"open: return 1",
		"write: arg0 == 1",
		"close: arg0 == 2; return 9",
		"dup: arg0 == 1 || arg0 == 2",
		"pipe: arg0 == 1 && arg1 == 2",
		"link: arg0 != 1 && arg1 != 2 || arg2 == 3",
		"unlink: arg0 != 1 || arg1 != 2 && arg2 == 3",
		"creat: arg0 & 0xf00 && arg1 & 0x0f0 && arg2 & 0x00f",
		"lseek: arg0 & 0x0f000f000f000f00 && arg1 & 0x00f000f000f000f0 && arg2 & 0x000f000f000f000f",
		"stat: arg0 != 0x0123456789abcdef && arg1 != 0x123456789abcdef0 || arg2 == 0x00f000f000000000",
		"fstat: arg0 != 0x0123456789abcdef || arg1 != 0x123456789abcdef0 && arg2 == 0x00f000f000000000",
	}
	tests := []struct {
		data SeccompData
		want uint32
	}{
		{call("fork"), retKill()},
		{call("read"), retAllow()},
		{call("open"), retErrno(1)},
		{call("write", 0), retKill()},
		{call("write", 1), retAllow()},
		{call("close", 1), retErrno(9)},
		{call("close", 2), retAllow()},
		{call("dup", 0), retKill()},
		{call("dup", 1), retAllow()},
		{call("dup", 2), retAllow()},
		{call("dup", 3), retKill()},
		{call("pipe", 1, 1), retKill()},
		{call("pipe", 1, 2), retAllow()},
		{call("pipe", 2, 1), retKill()},
		{call("pipe", 2, 2), retKill()},
		{call("link", 1, 2, 3), retAllow()},
		{call("link", 1, 2, 2), retKill()},
		{call("link", 2, 2, 2), retKill()},
		{call("link", 2, 1, 2), retAllow()},
		{call("unlink", 2, 1, 2), retAllow()},
		{call("unlink", 1, 1, 2), retKill()},
		{call("unlink", 1, 1, 3), retAllow()},
		{call("unlink", 1, 2, 3), retKill()},
		{call("creat", 0x100, 0x100, 0x101), retKill()},
		{call("creat", 0x200, 0x110, 0x101), retAllow()},
		{call("creat", 0x400, 0x110, 0x110), retKill()},
		{call("creat", 0x800, 0x110, 0x007), retAllow()},
		{call("lseek", 0x0100, 0x0100, 0x0101), retKill()},
		{call("lseek", 0x0200, 0x0110, 0x0101), retAllow()},
		{call("lseek", 0x0400, 0x0110, 0x0110), retKill()},
		{call("lseek", 0x0800, 0x0110, 0x0007), retAllow()},
		{call("lseek", 0x0100000000000000, 0x0100000000000000, 0x0101000000000000), retKill()},
		{call("lseek", 0x0200000000000000, 0x0110000000000000, 0x0101000000000000), retAllow()},
		{call("lseek", 0x0400000000000000, 0x0110000000000000, 0x0110000000000000), retKill()},
		{call("lseek", 0x0800000000000000, 0x0110000000000000, 0x0007000000000000), retAllow()},
		{call("stat", 0x0123456789abcdef, 0x123456789abcdef0, 0x00f000f000000000), retAllow()},
		{call("stat", 0x0123456789abcdef, 0x123456789abcdef0, 0x007000f000000000), retKill()},
		{call("stat", 0x0133456789abcdef, 0x123457789abcdef0, 0x007000f000000000), retAllow()},
		{call("stat", 0x0133456789abcdef, 0x123456789abcdef0, 0x007000f000000000), retKill()},
		{call("fstat", 0x0123456789abcdef, 0x123456789abcdef0, 0x00f000f000000000), retKill()},
		{call("fstat", 0x0133456789abcdef, 0x123456789abcdef0, 0x00f000f000000000), retAllow()},
		{call("fstat", 0x0123456789abcdef, 0x123457789abcdef0, 0x00f000f000000000), retAllow()},
		{call("fstat", 0x0123456789abcdef, 0x123457789abcdef0, 0x007000f000000000), retKill()},
	}

	ps, err := parseLines(file)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	bpf, err := compile(ps, true, bpfRet(retKill()))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	t.Logf("len(bpf) = %d", len(bpf))
	for _, test := range tests {
		if got := eval(bpf, test.data); got != test.want {
			t.Errorf("%s%#x = %#08x; want %#08x", syscallName[test.data.NR], test.data.Args, got, test.want)
		}
	}
}

func TestSupport(t *testing.T) {
	if err := CheckSupport(); err != nil {
		t.Error(err)
	}
}

func TestInstall(t *testing.T) {
	if !*testInstall {
		t.Skip("use -test_install (with -cpu=N) to enable.")
	}

	file := []string{
		"# open: return EPERM",
		"open: return 1",
		"# default: ALLOW",
	}

	ps, err := parseLines(file)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	bpf, err := compile(ps, nbits > 32, bpfRet(retAllow()))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if err = Install(bpf); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	N := runtime.GOMAXPROCS(0)
	opened := make(chan bool)
	for i := 0; i < N; i++ {
		go func() {
			if f, err := os.Open("/dev/null"); err != nil {
				t.Logf("open() failed: %v", err)
				opened <- false
			} else {
				t.Logf("open() succeeded")
				f.Close()
				opened <- true
			}
		}()
	}
	for i := 0; i < N; i++ {
		if <-opened {
			t.Fail()
		}
	}
}

func TestEndian(t *testing.T) {
	if !*testEndian {
		t.Skip("use -test_endian to enable.")
	}

	pass := syscall.EDOM
	fail := syscall.ERANGE
	name := map[error]string{
		pass: "pass",
		fail: "fail",
		nil:  "<nil>",
	}

	type seccomp [3]uintptr
	ps := []policy{
		{
			name: "seccomp",
			expr: orExpr{
				andExpr{
					argComp{0, "==", 0x0123456789abcdef},
				},
				andExpr{
					argComp{1, "!=", 0x01234567},
					argComp{2, "&", 0x01010101},
				},
			},
			then: bpfRet(retErrno(pass)),
			def:  ptr(bpfRet(retErrno(fail))),
		},
	}
	tests := []struct {
		args seccomp
		want error
	}{
		{seccomp{0x01234567, 0, 0}, fail},
		{seccomp{0x89abcdef, 0, 0}, pass},
		{seccomp{0, 0x01234567, 0}, fail},
		{seccomp{0, 0x12345678, 0}, fail},
		{seccomp{0, 0x01234567, 1}, fail},
		{seccomp{0, 0x12345678, 1}, pass},
	}
	call := func(args seccomp) error {
		if nr, ok := syscallNum["seccomp"]; ok {
			if _, _, e := syscall.Syscall(uintptr(nr), args[0], args[1], args[2]); e != 0 {
				return e
			}
		}
		return nil
	}

	bpf, err := compile(ps, false, bpfRet(retAllow()))
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if err = Install(bpf); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	for _, test := range tests {
		if got := call(test.args); got != test.want {
			t.Errorf("seccomp%#x = %v; want %v", test.args, name[got], name[test.want])
		}
	}
}
