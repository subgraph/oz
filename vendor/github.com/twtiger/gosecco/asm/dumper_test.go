package asm

import (
	"syscall"
	"testing"

	"github.com/twtiger/gosecco/compiler"

	"golang.org/x/sys/unix"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type DumperSuite struct{}

var _ = Suite(&DumperSuite{})

func (s *DumperSuite) Test_simpleDump(c *C) {
	inp := []unix.SockFilter{
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    0,
		},

		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   0,
			Jf:   8,
			K:    syscall.SYS_WRITE,
		},

		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_IMM,
			K:    12,
		},

		unix.SockFilter{
			Code: syscall.BPF_ALU | syscall.BPF_ADD | syscall.BPF_K,
			K:    4,
		},

		unix.SockFilter{
			Code: syscall.BPF_MISC | syscall.BPF_TAX,
		},

		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    0x14,
		},

		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   0,
			Jf:   3,
			K:    0,
		},

		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    0x10,
		},

		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_X,
			Jt:   0,
			Jf:   1,
			K:    0,
		},

		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    compiler.SECCOMP_RET_ALLOW,
		},

		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    compiler.SECCOMP_RET_KILL,
		},
	}

	res := Dump(inp)

	c.Assert(res, Equals, ""+
		`ld_abs	0
jeq_k	00	08	1
ld_imm	C
add_k	4
tax
ld_abs	14
jeq_k	00	03	0
ld_abs	10
jeq_x	00	01
ret_k	7FFF0000
ret_k	0
`)
}
