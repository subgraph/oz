package seccomp

import (
	"fmt"
)

// #include "linux/futex.h"
import "C"

func render_futex(pid int, args RegisterArgs) (string, error) {

	ops := map[uint]string{
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

	callrep := ""
	op := uint(args[1]) & 127
	opstr := ""

	for x, y := range ops {
		if uint(args[1])&x == x {
			opstr = y
		}
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
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, %d, %d)", uintptr(args[0]), opstr, int32(args[2]), args[3], args[5])
	default:
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, 0x%X, 0x%x, %d)", uintptr(args[0]), opstr, int32(args[2]), uintptr(args[3]), args[4], args[5])
	}
	return callrep, nil

}
