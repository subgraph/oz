package emulator

import (
	"syscall"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/twtiger/gosecco/data"

	. "gopkg.in/check.v1"
)

func ScenariosTest(t *testing.T) { TestingT(t) }

type ScenariosSuite struct{}

var _ = Suite(&ScenariosSuite{})

func (s *ScenariosSuite) Test_simpleBooleanReturnAllow(c *C) {
	data := data.SeccompWorkingMemory{Arch: 15, NR: 1}
	filters := []unix.SockFilter{
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    uint32(4),
		},
		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   uint8(0),
			Jf:   uint8(3),
			K:    uint32(15),
		},
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    uint32(0),
		},
		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   uint8(0),
			Jf:   uint8(1),
			K:    uint32(1),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0x7FFF0000),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0),
		},
	}

	res := Emulate(data, filters)
	c.Assert(res, Equals, uint32(0x7FFF0000))
}

func (s *ScenariosSuite) Test_simpleBooleanReturnKill(c *C) {
	data := data.SeccompWorkingMemory{Arch: 15, NR: 0}
	filters := []unix.SockFilter{
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    uint32(4),
		},
		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   uint8(0),
			Jf:   uint8(3),
			K:    uint32(15),
		},
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    uint32(0),
		},
		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   uint8(0),
			Jf:   uint8(1),
			K:    uint32(1),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0x7FFF0000),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0),
		},
	}

	res := Emulate(data, filters)
	c.Assert(res, Equals, uint32(0))
}

func (s *ScenariosSuite) Test_simpleComparisonWithSuccessCase(c *C) {
	data := data.SeccompWorkingMemory{Arch: 15, Args: [6]uint64{1, 0, 0, 0, 0, 0}}
	filters := []unix.SockFilter{
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    uint32(0x10),
		},
		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   uint8(0),
			Jf:   uint8(1),
			K:    uint32(1),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0x7FFF0000),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0),
		},
	}

	res := Emulate(data, filters)
	c.Assert(res, Equals, uint32(0x7FFF0000))
}

func (s *ScenariosSuite) Test_simpleComparisonWithFailureCase(c *C) {
	data := data.SeccompWorkingMemory{Arch: 15, Args: [6]uint64{0, 0, 0, 0, 0, 0}}
	filters := []unix.SockFilter{
		unix.SockFilter{
			Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_ABS,
			K:    uint32(0x14),
		},
		unix.SockFilter{
			Code: syscall.BPF_JMP | syscall.BPF_JEQ | syscall.BPF_K,
			Jt:   uint8(0),
			Jf:   uint8(1),
			K:    uint32(1),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0x7FFF0000),
		},
		unix.SockFilter{
			Code: syscall.BPF_RET | syscall.BPF_K,
			K:    uint32(0),
		},
	}

	res := Emulate(data, filters)
	c.Assert(res, Equals, uint32(0))
}

func (s *ScenariosSuite) Test_simpleAddition(c *C) {
	e := &emulator{
		data: data.SeccompWorkingMemory{Arch: 15, Args: [6]uint64{0, 0, 0, 0, 0, 0}},
		filters: []unix.SockFilter{
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_IMM,
				K:    uint32(4),
			},
			unix.SockFilter{
				Code: syscall.BPF_ST,
				K:    uint32(0),
			},
			unix.SockFilter{
				Code: syscall.BPF_LD | syscall.BPF_W | syscall.BPF_IMM,
				K:    uint32(3),
			},
			unix.SockFilter{
				Code: syscall.BPF_LDX | syscall.BPF_W | syscall.BPF_MEM,
				K:    uint32(0),
			},
			unix.SockFilter{
				Code: syscall.BPF_ALU | syscall.BPF_ADD | syscall.BPF_X,
			},
		},
	}

	e.next()
	e.next()
	e.next()
	e.next()
	e.next()

	c.Assert(e.A, Equals, uint32(7))
}
