package data

import "golang.org/x/sys/unix"

// SeccompWorkingMemory represents the piece of memory BPF is operating on when a program is running
type SeccompWorkingMemory struct {
	NR                 int32     // The system call number.
	Arch               uint32    // System call convention as an AUDIT_ARCH_* value.
	InstructionPointer uint64    // At the time of the system call.
	Args               [6]uint64 // System call arguments (always stored as 64-bit values).
}

// SockFprog contains a sock filter program
type SockFprog struct {
	Len    uint16           // Number of BPF machine instructions.
	Filter *unix.SockFilter // Pointer to the first instruction.
}
