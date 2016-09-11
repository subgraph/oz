package native

// #include <asm/unistd.h>
import "C"

// X32SyscallBit contains the bit that syscalls for the 32bit ABI will have set
const X32SyscallBit = uint32(C.__X32_SYSCALL_BIT)
