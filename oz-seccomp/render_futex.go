package seccomp

import (
	"fmt"
)

/* TODO Import from C headers instead of hardcoding these if not in golang syscall... */

const (
	FUTEX_WAIT            = 0
	FUTEX_WAKE            = 1
	FUTEX_FD              = 2
	FUTEX_REQUEUE         = 3
	FUTEX_CMP_REQUEUE     = 4
	FUTEX_WAKE_OP         = 5
	FUTEX_LOCK_PI         = 6
	FUTEX_UNLOCK_PI       = 7
	FUTEX_TRYLOCK_PI      = 8
	FUTEX_WAIT_BITSET     = 9
	FUTEX_WAKE_BITSET     = 10
	FUTEX_WAIT_REQUEUE_PI = 11
	FUTEX_CMP_REQUEUE_PI  = 12
	FUTEX_PRIVATE_FLAG    = 128
	FUTEX_CLOCK_REALTIME  = 256

	FUTEX_WAIT_PRIVATE        = (FUTEX_WAIT | FUTEX_PRIVATE_FLAG)
	FUTEX_WAKE_PRIVATE        = (FUTEX_WAKE | FUTEX_PRIVATE_FLAG)
	FUTEX_REQUEUE_PRIVATE     = (FUTEX_REQUEUE | FUTEX_PRIVATE_FLAG)
	FUTEX_CMP_REQUEUE_PRIVATE = (FUTEX_CMP_REQUEUE | FUTEX_PRIVATE_FLAG)
)

func render_futex(pid int, args RegisterArgs) (string, error) {

	ops := map[uint]string{
		FUTEX_WAIT:                             "FUTEX_WAIT",
		FUTEX_WAKE:                             "FUTEX_WAKE",
		FUTEX_FD:                               "FUTEX_FD",
		FUTEX_REQUEUE:                          "FUTEX_REQUEUE",
		FUTEX_CMP_REQUEUE:                      "FUTEX_CMP_REQUEUE",
		FUTEX_WAIT | FUTEX_PRIVATE_FLAG:        "FUTEX_WAIT_PRIVATE",
		FUTEX_WAKE | FUTEX_PRIVATE_FLAG:        "FUTEX_WAKE_PRIVATE",
		FUTEX_REQUEUE | FUTEX_PRIVATE_FLAG:     "FUTEX_REQUEUE_PRIVATE",
		FUTEX_CMP_REQUEUE | FUTEX_PRIVATE_FLAG: "FUTEX_CMP_REQUEUE_PRIVATE",
	}

	callrep := ""
	op := uint(args[1]) & 127
	opstr := ops[uint(args[1])]

	// TODO: lots

	if op == FUTEX_WAIT {
		callrep = fmt.Sprintf("futex(0x%X, %s, %d)", uintptr(args[0]), opstr, int32(args[2]))
	} else if op == FUTEX_WAKE {
		callrep = fmt.Sprintf("futex(0x%X, %s, %d)", uintptr(args[0]), opstr, int32(args[2]))
	} else if op == FUTEX_FD {
		callrep = fmt.Sprintf("futex(0x%X, %s, %d)", uintptr(args[0]), opstr, int32(args[2]))
	} else if op == FUTEX_WAKE_BITSET {
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, %d)", uintptr(args[0]), opstr, int32(args[2]), args[5])
	} else if op == FUTEX_WAIT_BITSET {
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, %d, %d", uintptr(args[0]), opstr, int32(args[2]), args[3], args[5])
	} else {
		callrep = fmt.Sprintf("futex(0x%X, %s, %d, 0x%X, 0x%x, %d)", uintptr(args[0]), opstr, int32(args[2]), uintptr(args[3]), args[4], args[5])
	}
	return callrep, nil

}
