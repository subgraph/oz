package emulator

import (
	"syscall"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/twtiger/gosecco/data"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type EmulatorSuite struct{}

var _ = Suite(&EmulatorSuite{})

func (s *EmulatorSuite) Test_simpleReturnK(c *C) {
	res := Emulate(data.SeccompWorkingMemory{}, []unix.SockFilter{
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(42),
		},
	})
	c.Assert(res, Equals, uint32(42))
}

func (s *EmulatorSuite) Test_simpleReturnX(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_RET | syscall.BPF_X,
				K:    uint32(42),
			},
		},
		pointer: 0,

		X: uint32(23),
	}

	res, _ := e.next()

	c.Assert(res, Equals, uint32(23))
}

func (s *EmulatorSuite) Test_loadValues(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{Arch: 15, Args: [6]uint64{0, 0, 0, 12423423, 0, 0}},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
				K:    uint32(4),
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
				K:    uint32(40),
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_IND,
				K:    uint32(2),
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_IND,
				K:    uint32(38),
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_LEN,
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_IMM,
				K:    uint32(23),
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_MEM,
				K:    0,
			},

			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_MEM,
				K:    2,
			},
		},
		pointer: 0,
		X:       uint32(2),
		M:       [16]uint32{2, 3, 4},
	}

	e.next()

	c.Assert(e.A, Equals, uint32(15))

	e.next()

	c.Assert(e.A, Equals, uint32(12423423))

	e.next()

	c.Assert(e.A, Equals, uint32(15))

	e.next()

	c.Assert(e.A, Equals, uint32(12423423))

	e.next()

	c.Assert(e.A, Equals, uint32(64))

	e.next()

	c.Assert(e.A, Equals, uint32(23))

	e.next()

	c.Assert(e.A, Equals, uint32(2))

	e.next()

	c.Assert(e.A, Equals, uint32(4))
}

func loadAbs(k uint32) unix.SockFilter {
	return unix.SockFilter{
		Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
		K:    k,
	}
}

func (s *EmulatorSuite) Test_loadWorkingMemory(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{NR: 15, InstructionPointer: 45365364654, Arch: 15, Args: [6]uint64{123234, 5465645, 12132, 12423423, 7766, 12124}},
		filters: []unix.SockFilter{
			loadAbs(0),
			loadAbs(4),
			loadAbs(8),
			loadAbs(12),
			loadAbs(16),
			loadAbs(20),
			loadAbs(24),
			loadAbs(28),
			loadAbs(32),
			loadAbs(36),
			loadAbs(40),
			loadAbs(44),
			loadAbs(48),
			loadAbs(52),
			loadAbs(56),
			loadAbs(60),
			loadAbs(3),
		},
		pointer: 0,
	}

	e.next()
	c.Assert(e.A, Equals, uint32(0xF))

	e.next()
	c.Assert(e.A, Equals, uint32(0xF))

	e.next()
	c.Assert(e.A, Equals, uint32(0xA))

	e.next()
	c.Assert(e.A, Equals, uint32(0x87AE))

	e.next()
	c.Assert(e.A, Equals, uint32(0x1E162))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0x53662d))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0x2F64))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0xbd90ff))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0x1e56))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0x2f5c))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0))

	e.next()
	c.Assert(e.A, Equals, uint32(0))
}

func (s *EmulatorSuite) Test_loadValuesIntoX(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_LDX | syscall.BPF_IMM,
				K:    uint32(234),
			},
			unix.SockFilter{
				Code: syscall.BPF_LDX | syscall.BPF_W | syscall.BPF_LEN,
			},
			unix.SockFilter{
				Code: syscall.BPF_LDX | syscall.BPF_W | syscall.BPF_MEM,
				K:    1,
			},
		},
		pointer: 0,
		M:       [16]uint32{2, 3, 4},
	}

	e.next()

	c.Assert(e.X, Equals, uint32(234))

	e.next()

	c.Assert(e.X, Equals, uint32(64))

	e.next()

	c.Assert(e.X, Equals, uint32(3))
}

func aluAndK(c *C, op uint16, a, k, expected uint32) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_ALU | syscall.BPF_K | op,
				K:    k,
			},
		},
		pointer: 0,
		A:       a,
	}

	e.next()

	c.Assert(e.A, Equals, expected)
}

func aluAndX(c *C, op uint16, a, k, expected uint32) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_ALU | syscall.BPF_X | op,
			},
		},
		pointer: 0,
		A:       a,
		X:       k,
	}

	e.next()

	c.Assert(e.A, Equals, expected)
}

func (s *EmulatorSuite) Test_aluAandK(c *C) {
	aluAndK(c, syscall.BPF_ADD, 15, 42, 57)
	aluAndK(c, syscall.BPF_SUB, 10, 3, 7)
	aluAndK(c, syscall.BPF_MUL, 10, 3, 30)
	aluAndK(c, syscall.BPF_DIV, 10, 3, 3)
	aluAndK(c, syscall.BPF_AND, 32425, 1211, 32425&1211)
	aluAndK(c, syscall.BPF_OR, 32425, 1211, 32425|1211)
	aluAndK(c, BPF_XOR, 32425, 1211, 32425^1211)
	aluAndK(c, syscall.BPF_LSH, 10, 3, 80)
	aluAndK(c, syscall.BPF_RSH, 80, 3, 10)
	aluAndK(c, BPF_MOD, 10, 3, 1)
	aluAndK(c, syscall.BPF_NEG, 80, 0, 0xFFFFFFB0)
}

func (s *EmulatorSuite) Test_aluAandX(c *C) {
	aluAndX(c, syscall.BPF_ADD, 15, 42, 57)
	aluAndX(c, syscall.BPF_SUB, 10, 3, 7)
	aluAndX(c, syscall.BPF_MUL, 10, 3, 30)
	aluAndX(c, syscall.BPF_DIV, 10, 3, 3)
	aluAndX(c, syscall.BPF_AND, 32425, 1211, 32425&1211)
	aluAndX(c, syscall.BPF_OR, 32425, 1211, 32425|1211)
	aluAndX(c, BPF_XOR, 32425, 1211, 32425^1211)
	aluAndX(c, syscall.BPF_LSH, 10, 3, 80)
	aluAndX(c, syscall.BPF_RSH, 80, 3, 10)
	aluAndX(c, BPF_MOD, 10, 3, 1)
}

func (s *EmulatorSuite) Test_misc(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_MISC | syscall.BPF_TAX,
			},
		},
		pointer: 0,
		A:       42,
		X:       23,
	}

	e.next()

	c.Assert(e.X, Equals, uint32(42))

	e = &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_MISC | syscall.BPF_TXA,
			},
		},
		pointer: 0,
		A:       42,
		X:       23,
	}

	e.next()

	c.Assert(e.A, Equals, uint32(23))
}

func (s *EmulatorSuite) Test_simpleJump(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JA,
				K:    42,
			},
		},
		pointer: 0,
	}

	e.next()

	c.Assert(e.pointer, Equals, uint32(43))
}

func (s *EmulatorSuite) Test_simpleJgtK(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JGT | syscall.BPF_K,
				Jt:   1,
				Jf:   2,
				K:    5,
			},
		},
		pointer: 0,
		A:       0,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 6

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJgeK(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JGE | syscall.BPF_K,
				Jt:   1,
				Jf:   2,
				K:    5,
			},
		},
		pointer: 0,
		A:       0,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 5

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJeqK(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
				Jt:   1,
				Jf:   2,
				K:    3,
			},
		},
		pointer: 0,
		A:       0,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 3

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJsetK(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JSET | syscall.BPF_K,
				Jt:   1,
				Jf:   2,
				K:    8,
			},
		},
		pointer: 0,
		A:       7,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 15

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJgtX(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JGT | syscall.BPF_X,
				Jt:   1,
				Jf:   2,
			},
		},
		pointer: 0,
		A:       0,
		X:       5,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 6

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJgeX(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JGE | syscall.BPF_X,
				Jt:   1,
				Jf:   2,
			},
		},
		pointer: 0,
		A:       0,
		X:       5,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 5

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJeqX(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_X,
				Jt:   1,
				Jf:   2,
			},
		},
		pointer: 0,
		A:       0,
		X:       3,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 3

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_simpleJsetX(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_JMP | syscall.BPF_JSET | syscall.BPF_X,
				Jt:   1,
				Jf:   2,
			},
		},
		pointer: 0,
		A:       7,
		X:       8,
	}

	e.next()
	c.Assert(e.pointer, Equals, uint32(3))

	e.pointer = 0
	e.A = 15

	e.next()
	c.Assert(e.pointer, Equals, uint32(2))
}

func (s *EmulatorSuite) Test_storeAInScratchMemory(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_ST,
				K:    0,
			},
			unix.SockFilter{
				Code: syscall.BPF_ST,
				K:    1,
			},
		},
		pointer: 0,
		A:       5,
	}

	e.next()

	c.Assert(e.M[0], Equals, uint32(5))

	e.A = 4
	e.next()

	c.Assert(e.M[1], Equals, uint32(4))
}

func (s *EmulatorSuite) Test_storeXInScratchMemory(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_STX,
				K:    0,
			},
			unix.SockFilter{
				Code: syscall.BPF_STX,
				K:    1,
			},
		},
		pointer: 0,
		X:       5,
	}

	e.next()

	c.Assert(e.M[0], Equals, uint32(5))

	e.X = 4
	e.next()

	c.Assert(e.M[1], Equals, uint32(4))
}
