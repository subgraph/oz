package seccomp

import (
	"syscall"
	"fmt"
)

type RegisterArgs []uint64

func getSyscallRegisterArgs(regs syscall.PtraceRegs) RegisterArgs {
	return []uint64{regs.Rdi, regs.Rsi, regs.Rdx, regs.Rcx, regs.R8, regs.R9}
}

func getSyscallNumber(regs syscall.PtraceRegs) int {
	return int(regs.Orig_rax)
}

func renderSyscallBasic(pid int, systemcall SystemCall, regs syscall.PtraceRegs) string {

	var callrep string = fmt.Sprintf("%s(", systemcall.name)
	var reg uint64 = 0

	for arg := range systemcall.args {

		if systemcall.args[arg] == 0 {
			break
		}

		if arg > 0 {
			callrep += fmt.Sprintf(",")
		}

		switch arg {
		case 0:
			reg = regs.Rdi
		case 1:
			reg = regs.Rsi
		case 2:
			reg = regs.Rdx
		case 3:
			reg = regs.Rcx
		case 4:
			reg = regs.R8
		case 5:
			reg = regs.R9
		}
		if systemcall.args[arg] == STRINGARG {
			str, err := readStringArg(pid, uintptr(reg))
			if err != nil {
				log.Error("Error: %v", err)
			} else {
				callrep += fmt.Sprintf("\"%s\"", str)
			}
		} else if systemcall.args[arg] == INTARG {
			callrep += fmt.Sprintf("%d", uint64(reg))
		} else {
			/* Stringify pointers in writes to stdout/stderr */
			write, err := syscallByName("write")
			if err != nil {
				log.Error("Error: %v", err)
			}
			if systemcall.num == write.num && (regs.Rdi == uint64(syscall.Stdout) || regs.Rdi == uint64(syscall.Stderr)) {
				str, err := readStringArg(pid, uintptr(reg))
				if err != nil {
					log.Error("Error %v", err)
				} else {
					if isPrintableASCII(str) == true {
						callrep += fmt.Sprintf("\"%s\"", str)
					} else {
						callrep += fmt.Sprintf("0x%X", uintptr(reg))
					}
				}
			} else {
				callrep += fmt.Sprintf("0x%X", uintptr(reg))
			}
		}

	}
	callrep += ")"
	return fmt.Sprintf("==============================================\nseccomp hit on sandbox pid %v (%v) syscall %v (%v): \n\n%s\nI ==============================================\n\n", pid, getProcessCmdLine(pid), systemcall.name, regs.Orig_rax, callrep)
}
