package seccomp

import (
	"fmt"
)

// #include "asm-generic/fcntl.h"
import "C"

var openflags = map[int32]string{
	C.O_RDONLY: "O_RDONLY",
	C.O_WRONLY: "O_WRONLY",
	C.O_RDWR:   "O_RDWR",
}

var creatflags = map[uint]string{
	C.O_CLOEXEC:   "O_CLOEXEC",
	C.O_CREAT:     "O_CREAT",
	C.O_DIRECTORY: "O_DIRECTORY",
	C.O_EXCL:      "O_EXCL",
	C.O_NOCTTY:    "O_NOCTTY",
	C.O_NOFOLLOW:  "O_NOFOLLOW",
	C.O_TRUNC:     "O_TRUNC",
	//	syscall.O_TTY_INIT:  "O_TTY_INIT",
	C.O_TMPFILE: "O_TMPFILE",
}

func render_openat(pid int, args RegisterArgs) (string, error) {

	fd := int32(args[0])
	flagval := int32(args[2])
	mode := uint32(args[3])
	path, err := readStringArg(pid, uintptr(args[1]))

	if err != nil {
		return "", err
	}

	openflagstr := ""
	fdstr := ""

	if (flagval & C.O_RDONLY) == C.O_RDONLY {
		openflagstr += "O_RDONLY"
	} else if (flagval & C.O_WRONLY) == C.O_WRONLY {
		openflagstr += "O_WRONLY"
	} else if (flagval & C.O_RDWR) == C.O_RDWR {
		openflagstr += "O_RDWR"
	}

	tmp := renderFlags(creatflags, uint(flagval))
	if tmp != "" {
		openflagstr += "|"
		openflagstr += tmp
	}

	if fd == -100 {
		fdstr = "AT_FDCWD"
	} else {
		fdstr = fmt.Sprintf("%d", fd)	
	}

	if (flagval & C.O_CREAT) == C.O_CREAT) || (flagval & C.O_TMPFILE) == C.O_TMPFILE) {
		callrep := fmt.Sprintf("openat(%s, \"%s\", %s, %d)", fdstr, path, openflagstr, mode)
	}
	else {
		                callrep := fmt.Sprintf("openat(%s, \"%s\", %s)", fdstr, path, openflagstr)
			}

	return callrep, nil
}
