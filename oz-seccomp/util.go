package seccomp

import (
	"encoding/binary"
	"errors"
	"io/ioutil"
	"strconv"
	"unsafe"
	"fmt"
	"syscall"
)

func readStringArg(pid int, addr uintptr) (s string, err error) {
	buf := make([]byte, unsafe.Sizeof(addr))
	done := false
	err = nil
	for done == false {
		_, err := syscall.PtracePeekText(pid, addr, buf)
		if err != nil {
			fmt.Printf("Error (ptrace): %v\n", err)
		} else {
			for b := range buf {
				if buf[b] == 0 {
					done = true
					break
				} else {
					s += string(buf[b])
					if len(s) > 90 {
						s += "..."
						break
					}
				}
			}
		}
		addr += unsafe.Sizeof(addr)
	}
	return s, nil
}

func readUIntArg(pid int, addr uintptr) (uint64, error) {
	buf := make([]byte, unsafe.Sizeof(addr))

	_, err := syscall.PtracePeekText(pid, addr, buf)
	if err != nil {
		fmt.Printf("Error (ptrace): %v\n", err)
		return 0, err
	} else {
		i := binary.LittleEndian.Uint64(buf)
		return i, nil
	}
	return 0, errors.New("Error.")
}

func readPtrArg(pid int, addr uintptr) (uintptr, error) {

	buf := make([]byte, unsafe.Sizeof(addr))

	_, err := syscall.PtracePeekText(pid, addr, buf)
	if err != nil {
		fmt.Printf("Error (ptrace): %v\n", err)
		return 0, err
	} else {
		i := binary.LittleEndian.Uint64(buf)
		return uintptr(i), nil
	}
	return 0, nil
}

func syscallByNum(num int) (s SystemCall, err error) {
	var q SystemCall = SystemCall{"", "", -1, []int{0, 0, 0, 0, 0, 0}}
	for i := range syscalls {
		if syscalls[i].num == num {
			q = syscalls[i]
			return q, nil
		}
	}
	return q, errors.New("System call not found.\n")
}

func syscallByName(name string) (s SystemCall, err error) {
	var q SystemCall = SystemCall{"", "", -1, []int{0, 0, 0, 0, 0, 0}}
	for i := range syscalls {
		if syscalls[i].name == name {
			q = syscalls[i]
			return q, nil
		}
	}
	return q, errors.New("System call not found.\n")
}

func isPrintableASCII(s string) bool {
	for _, x := range s {
		if x > 127 {
			return false
		}
	}
	return true
}

func getProcessCmdLine(pid int) string {
	path := "/proc/" + strconv.Itoa(pid) + "/cmdline"
	cmdline, err := ioutil.ReadFile(path)
	for b := range cmdline {
		if b <= (len(cmdline) - 1) {
			if cmdline[b] == 0x00 {
				cmdline[b] = 0x20
			}
		}
	}
	if err != nil {
		log.Error("Error (read): %v", err)
		return "unknown"
	}
	return string(cmdline)
}
