package seccomp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

// #include "linux/futex.h"
import "C"

func render_futex(pid int, args RegisterArgs) (string, error) {

	ops := map[uint32]string{
		C.FUTEX_WAIT:                    "FUTEX_WAIT",
		C.FUTEX_WAKE:                    "FUTEX_WAKE",
		C.FUTEX_FD:                      "FUTEX_FD",
		C.FUTEX_REQUEUE:                 "FUTEX_REQUEUE",
		C.FUTEX_CMP_REQUEUE:             "FUTEX_CMP_REQUEUE",
		C.FUTEX_WAKE_OP:                 "FUTEX_WAKE_OP",
		C.FUTEX_LOCK_PI:                 "FUTEX_LOCK_PI",
		C.FUTEX_UNLOCK_PI:               "FUTEX_UNLOCK_PI",
		C.FUTEX_TRYLOCK_PI:              "FUTEX_TRYLOCK_PI",
		C.FUTEX_WAIT_BITSET:             "FUTEX_WAIT_BITSET",
		C.FUTEX_WAKE_BITSET:             "FUTEX_WAKE_BITSET",
		C.FUTEX_WAIT_REQUEUE_PI:         "FUTEX_WAIT_REQUEUE_PI",
		C.FUTEX_CMP_REQUEUE_PI:          "FUTEX_CMP_REQUEUE_PI",
		C.FUTEX_WAIT_PRIVATE:            "FUTEX_WAIT_PRIVATE",
		C.FUTEX_WAKE_PRIVATE:            "FUTEX_WAKE_PRIVATE",
		C.FUTEX_REQUEUE_PRIVATE:         "FUTEX_REQUEUE_PRIVATE",
		C.FUTEX_CMP_REQUEUE_PRIVATE:     "FUTEX_CMP_REQUEUE_PRIVATE",
		C.FUTEX_WAKE_OP_PRIVATE:         "FUTEX_WAKE_OP_PRIVATE",
		C.FUTEX_LOCK_PI_PRIVATE:         "FUTEX_LOCK_PI_PRIVATE",
		C.FUTEX_UNLOCK_PI_PRIVATE:       "FUTEX_UNLOCK_PI_PRIVATE",
		C.FUTEX_TRYLOCK_PI_PRIVATE:      "FUTEX_TRYLOCK_PI_PRIVATE",
		C.FUTEX_WAIT_BITSET_PRIVATE:     "FUTEX_WAIT_BITSET_PRIVATE",
		C.FUTEX_WAKE_BITSET_PRIVATE:     "FUTEX_WAKE_BITSET_PRIVATE",
		C.FUTEX_WAIT_REQUEUE_PI_PRIVATE: "FUTEX_WAIT_REQUEUE_PI_PRIVATE",
		C.FUTEX_CMP_REQUEUE_PI_PRIVATE:  "FUTEX_CMP_REQUEUE_PI_PRIVATE",
	}
	wakeops := map[uint32]string{
		C.FUTEX_OP_SET:  "FUTEX_OP_SET",
		C.FUTEX_OP_ADD:  "FUTEX_OP_ADD",
		C.FUTEX_OP_OR:   "FUTEX_OP_OR",
		C.FUTEX_OP_ANDN: "FUTEX_OP_ANDN",
		C.FUTEX_OP_XOR:  "FUTEX_OP_XOR",
	}
	cmps := map[uint32]string{
		C.FUTEX_OP_CMP_EQ: "FUTEX_OP_CMP_EQ",
		C.FUTEX_OP_CMP_NE: "FUTEX_OP_CMP_NE",
		C.FUTEX_OP_CMP_LT: "FUTEX_OP_CMP_LT",
		C.FUTEX_OP_CMP_LE: "FUTEX_OP_CMP_LE",
		C.FUTEX_OP_CMP_GT: "FUTEX_OP_CMP_GT",
		C.FUTEX_OP_CMP_GE: "FUTEX_OP_CMP_GE",
	}

	callrep := ""
	opstr := ""

	op := uint32(args[1]) & 127
	if (uint32(args[1]) & C.FUTEX_PRIVATE_FLAG) == C.FUTEX_PRIVATE_FLAG {
		opstr = ops[op|C.FUTEX_PRIVATE_FLAG]
	} else {
		opstr = ops[op]
	}

	if opstr == "" {
		opstr = fmt.Sprintf("(FUTEX_??? %d %d %s)", args[1], op, ops[op])
	}

	if (args[1] & C.FUTEX_CLOCK_REALTIME) == C.FUTEX_CLOCK_REALTIME {
		opstr += "|FUTEX_CLOCK_REALTIME"
	}

	// TODO: lots

	switch op {
	case C.FUTEX_WAIT:
		callrep = fmt.Sprintf("futex(0x%X, %s, %d)", uintptr(args[0]), opstr, int32(args[2]))
	case C.FUTEX_WAKE:
		callrep = fmt.Sprintf("futex(0x%X, %s, %d)", uintptr(args[0]), opstr, int32(args[2]))
	case C.FUTEX_FD:
		callrep = fmt.Sprintf("futex(0x%X, %s, %d)", uintptr(args[0]), opstr, int32(args[2]))
	case C.FUTEX_WAKE_BITSET:
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, %d)", uintptr(args[0]), opstr, int32(args[2]), args[5])
	case C.FUTEX_WAIT_BITSET:
		var t syscall.Timespec
		if (args[3]) != 0 {
			buf, err := readBytesArg(pid, int(unsafe.Sizeof(t)), uintptr(args[3]))
			if err != nil {
				return "", err
			}
			b := bytes.NewBuffer(buf)
			binary.Read(b, binary.LittleEndian, &t)
		}
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, {%d, %d}, %x)", uintptr(args[0]), opstr, int32(args[2]), t.Sec, t.Nsec, uint32(args[5]))
	case C.FUTEX_WAKE_OP:
		wakeopstr := "{"
		wakeop := uint32(args[5])
		if ((wakeop >> 28) & C.FUTEX_OP_OPARG_SHIFT) == C.FUTEX_OP_OPARG_SHIFT {
			wakeopstr += "FUTEX_OP_OPARG_SHIFT|"
		}
		wakeopstr += wakeops[(wakeop>>28)&0x7]
		wakeopstr += fmt.Sprintf(", %d, ", (wakeop>>12)&0xfff)
		if (uint(wakeop>>24) & C.FUTEX_OP_OPARG_SHIFT) == C.FUTEX_OP_OPARG_SHIFT {
			wakeopstr += "FUTEX_OP_OPARG_SHIFT|"
		}
		wakeopstr += cmps[(wakeop>>24)&0x7]
		wakeopstr += fmt.Sprintf(", %d}", wakeop&0xfff)
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, 0x%X, %s)", uintptr(args[0]), opstr, int32(args[2]), uintptr(args[3]), wakeopstr)
	default:
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, 0x%X, 0x%x, %d)", uintptr(args[0]), opstr, int32(args[2]), uintptr(args[3]), args[4], args[5])
	}
	return callrep, nil

}
