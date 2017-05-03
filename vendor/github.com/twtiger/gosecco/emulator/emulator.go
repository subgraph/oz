package emulator

import (
	"fmt"
	"log"
	"syscall"

	"github.com/twtiger/gosecco/data"

	"golang.org/x/sys/unix"
)

func init() {
	log.SetFlags(0)
}

// Emulate will execute a seccomp filter program against the given working memory.
func Emulate(d data.SeccompWorkingMemory, filters []unix.SockFilter) uint32 {
	e := &emulator{data: d, filters: filters, pointer: 0}
	for {
		val, finished := e.next()
		if finished {
			return val
		}
	}
}

type emulator struct {
	data    data.SeccompWorkingMemory
	filters []unix.SockFilter
	pointer uint32

	X uint32
	A uint32
	M [syscall.BPF_MEMWORDS]uint32
}

func bpfClass(code uint16) uint16 {
	return code & 0x07
}

func bpfSize(code uint16) uint16 {
	return code & 0x18
}

func bpfMode(code uint16) uint16 {
	return code & 0xe0
}

func bpfOp(code uint16) uint16 {
	return code & 0xf0
}
func bpfMiscOp(code uint16) uint16 {
	return code & 0xf8
}

func bpfSrc(code uint16) uint16 {
	return code & 0x08
}

func (e *emulator) execRet(current unix.SockFilter) (uint32, bool) {
	switch bpfSrc(current.Code) {
	case syscall.BPF_K:
		return current.K, true
	case syscall.BPF_X:
		return e.X, true
	default:
		panic(fmt.Sprintf("Invalid ret source: %d", bpfSrc(current.Code)))
	}
	return 0, true
}

func (e *emulator) getFromWorkingMemory(ix uint32) uint32 {
	switch ix {
	case 0:
		return uint32(e.data.NR)
	case 4:
		return e.data.Arch
	case 8:
		return uint32(e.data.InstructionPointer >> 32)
	case 12:
		return uint32(e.data.InstructionPointer & 0xFFFF)
	case 16:
		return uint32(e.data.Args[0] & 0xFFFFFFFF)
	case 20:
		return uint32(e.data.Args[0] >> 32)
	case 24:
		return uint32(e.data.Args[1] & 0xFFFFFFFF)
	case 28:
		return uint32(e.data.Args[1] >> 32)
	case 32:
		return uint32(e.data.Args[2] & 0xFFFFFFFF)
	case 36:
		return uint32(e.data.Args[2] >> 32)
	case 40:
		return uint32(e.data.Args[3] & 0xFFFFFFFF)
	case 44:
		return uint32(e.data.Args[3] >> 32)
	case 48:
		return uint32(e.data.Args[4] & 0xFFFFFFFF)
	case 52:
		return uint32(e.data.Args[4] >> 32)
	case 56:
		return uint32(e.data.Args[5] & 0xFFFFFFFF)
	case 60:
		return uint32(e.data.Args[5] >> 32)
	default:
		return 0
	}
}

func (e *emulator) loadFromWorkingMemory(ix uint32) {
	e.A = e.getFromWorkingMemory(ix)
}

func (e *emulator) execLd(current unix.SockFilter) (uint32, bool) {
	cd := current.Code

	if bpfSize(cd) != syscall.BPF_W {
		panic("Invalid code, we can't load smaller values than wide ones")
	}

	switch bpfMode(cd) {
	case syscall.BPF_ABS:
		e.loadFromWorkingMemory(current.K)
	case syscall.BPF_IND:
		e.loadFromWorkingMemory(e.X + current.K)
	case syscall.BPF_LEN:
		e.A = uint32(64)
	case syscall.BPF_IMM:
		e.A = current.K
	case syscall.BPF_MEM:
		if current.K < syscall.BPF_MEMWORDS {
			e.A = e.M[current.K]
		} else {
			panic(fmt.Sprintf("Index out of range: %d greater than MEMWORDS %d", current.K, syscall.BPF_MEMWORDS))
		}
	default:
		panic(fmt.Sprintf("Invalid mode: %d", bpfMode(cd)))
	}
	return 0, false
}

func (e *emulator) execLdx(current unix.SockFilter) (uint32, bool) {
	cd := current.Code

	if bpfSize(cd) != syscall.BPF_W {
		panic("Invalid code, we can't load smaller values than wide ones")
	}

	switch bpfMode(cd) {
	case syscall.BPF_LEN:
		e.X = uint32(64)
	case syscall.BPF_IMM:
		e.X = current.K
	case syscall.BPF_MEM:
		if current.K < syscall.BPF_MEMWORDS {
			e.X = e.M[current.K]
		} else {
			panic(fmt.Sprintf("Index out of range: %d greater than MEMWORDS", current.K, syscall.BPF_MEMWORDS))
		}
	default:
		panic(fmt.Sprintf("Invalid mode: %d", bpfMode(cd)))
	}
	return 0, false
}

// BPF_MOD is BPF_MOD - it is supported in Linux from v3.7+, but not in go's syscall...
const BPF_MOD = 0x90

// BPF_XOR is BPF_XOR - it is supported in Linux from v3.7+, but not in go's syscall...
const BPF_XOR = 0xa0

func (e *emulator) execAlu(current unix.SockFilter) (uint32, bool) {
	cd := current.Code

	right := uint32(0)

	switch bpfSrc(cd) {
	case syscall.BPF_K:
		right = current.K
	case syscall.BPF_X:
		right = e.X
	default:
		panic(fmt.Sprintf("Invalid source for right hand side of operation: %d", bpfSrc(cd)))
	}

	switch bpfOp(cd) {
	case syscall.BPF_ADD:
		e.A += right
	case syscall.BPF_SUB:
		e.A -= right
	case syscall.BPF_MUL:
		e.A *= right
	case syscall.BPF_DIV:
		e.A /= right
	case syscall.BPF_AND:
		e.A &= right
	case syscall.BPF_OR:
		e.A |= right
	case BPF_XOR:
		e.A ^= right
	case syscall.BPF_LSH:
		e.A <<= right
	case syscall.BPF_RSH:
		e.A >>= right
	case BPF_MOD:
		e.A %= right
	case syscall.BPF_NEG:
		e.A = -e.A
	default:
		panic(fmt.Sprintf("Invalid op: %d", bpfOp(cd)))
	}
	return 0, false
}

func (e *emulator) execMisc(current unix.SockFilter) (uint32, bool) {
	cd := current.Code

	switch bpfMiscOp(cd) {
	case syscall.BPF_TAX:
		e.X = e.A
	case syscall.BPF_TXA:
		e.A = e.X
	default:
		panic(fmt.Sprintf("Invalid op: %d", bpfMiscOp(cd)))
	}
	return 0, false
}

func (e *emulator) execJmp(current unix.SockFilter) (uint32, bool) {
	cd := current.Code

	right := uint32(0)
	switch bpfSrc(cd) {
	case syscall.BPF_K:
		right = current.K
	case syscall.BPF_X:
		right = e.X
	default:
		panic(fmt.Sprintf("Invalid source for right hand side of operation: %d", bpfSrc(cd)))
	}

	switch bpfOp(cd) {
	case syscall.BPF_JA:
		e.pointer += current.K
	case syscall.BPF_JGT:
		if e.A > right {
			e.pointer += uint32(current.Jt)
		} else {
			e.pointer += uint32(current.Jf)
		}
	case syscall.BPF_JGE:
		if e.A >= right {
			e.pointer += uint32(current.Jt)
		} else {
			e.pointer += uint32(current.Jf)
		}
	case syscall.BPF_JEQ:
		if e.A == right {
			e.pointer += uint32(current.Jt)
		} else {
			e.pointer += uint32(current.Jf)
		}
	case syscall.BPF_JSET:
		if e.A&right != 0 {
			e.pointer += uint32(current.Jt)
		} else {
			e.pointer += uint32(current.Jf)
		}
	default:
		panic(fmt.Sprintf("Invalid op: %d", bpfOp(cd)))
	}
	return 0, false
}

func (e *emulator) execStore(current unix.SockFilter) (uint32, bool) {
	switch bpfClass(current.Code) {
	case syscall.BPF_ST:
		e.M[current.K] = e.A
		return 0, false
	case syscall.BPF_STX:
		e.M[current.K] = e.X
		return 0, false
	default:
		panic(fmt.Sprintf("Invalid op: %d", bpfClass(current.Code)))
	}
	return 0, false
}

func (e *emulator) next() (uint32, bool) {
	if e.pointer >= uint32(len(e.filters)) {
		return 0, true
	}

	current := e.filters[e.pointer]
	e.pointer++
	switch bpfClass(current.Code) {
	case syscall.BPF_RET:
		return e.execRet(current)
	case syscall.BPF_LD:
		return e.execLd(current)
	case syscall.BPF_LDX:
		return e.execLdx(current)
	case syscall.BPF_ALU:
		return e.execAlu(current)
	case syscall.BPF_MISC:
		return e.execMisc(current)
	case syscall.BPF_JMP:
		return e.execJmp(current)
	case syscall.BPF_ST:
		return e.execStore(current)
	case syscall.BPF_STX:
		return e.execStore(current)
	}

	return 0, true
}
