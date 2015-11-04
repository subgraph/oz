package seccomp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// #include "asm-generic/fcntl.h"
import "C"

var whences = map[int16]string{
	int16(os.SEEK_SET): "SEEK_SET",
	int16(os.SEEK_CUR): "SEEK_CUR",
	int16(os.SEEK_END): "SEEK_END",
}

var fcntllock = map[int16]string{
	C.F_RDLCK: "F_RDLCK",
	C.F_WRLCK: "F_WRLCK",
	C.F_UNLCK: "F_UNLCK",
}

var fcntlcmds = map[int32]string{
	C.F_DUPFD:         "F_DUPFD",
	C.F_GETFD:         "F_GETFD",
	C.F_SETFD:         "F_SETFD",
	C.F_GETFL:         "F_GETFL",
	C.F_SETFL:         "F_SETFL",
	C.F_GETLK:         "F_GETLK",
	C.F_SETLK:         "F_SETLK",
	C.F_SETLKW:        "F_SETLKW",
	C.F_SETOWN:        "F_SETOWN",
	C.F_GETOWN:        "F_GETOWN",
	C.F_SETSIG:        "F_SETSIG",
	C.F_GETSIG:        "F_GETSIG",
	C.F_GETLK64:       "F_GETLK64",
	C.F_SETLK64:       "F_SETLK64",
	C.F_SETLKW64:      "F_SETLKW64",
	C.F_SETOWN_EX:     "F_SETOWN_EX",
	C.F_GETOWN_EX:     "F_GETOWN_EX",
	C.F_GETOWNER_UIDS: "F_GETOWNER_UIDS",
}

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
	callrep := ""

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

	if ((flagval & C.O_CREAT) == C.O_CREAT) || ((flagval & C.O_TMPFILE) == C.O_TMPFILE) {
		callrep = fmt.Sprintf("openat(%s, \"%s\", %s, %#o)", fdstr, path, openflagstr, mode)
	} else {
		callrep = fmt.Sprintf("openat(%s, \"%s\", %s)", fdstr, path, openflagstr)
	}

	return callrep, nil
}

func render_open(pid int, args RegisterArgs) (string, error) {

	path, err := readStringArg(pid, uintptr(args[0]))
	if err != nil {
		return "", err
	}
	flagval := int32(args[1])
	mode := int32(args[2])
	openflagstr := ""
	callrep := ""

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

	if ((flagval & C.O_CREAT) == C.O_CREAT) || ((flagval & C.O_TMPFILE) == C.O_TMPFILE) {
		callrep = fmt.Sprintf("open(\"%s\", %s, %#o)", path, openflagstr, mode)
	} else {
		callrep = fmt.Sprintf("open(\"%s\", %s)", path, openflagstr)
	}

	return callrep, nil

}

func render_mkdir(pid int, args RegisterArgs) (string, error) {

	path, err := readStringArg(pid, uintptr(args[0]))
	if err != nil {
		return "", err
	}
	mode := int32(args[1])

	callrep := fmt.Sprintf("mkdir(\"%s\", %#o)", path, mode)

	return callrep, nil
}

func render_pipe(pid int, args RegisterArgs) (string, error) {

	buf, err := readBytesArg(pid, 8, uintptr(args[0]))
	fd1 := bytestoint32(buf[0:3])
	fd2 := bytestoint32(buf[4:7])

	callrep := fmt.Sprintf("pipe([%d, %d])", fd1, fd2)

	if err != nil {
		return "", err
	}
	return callrep, nil
}

func render_fcntl(pid int, args RegisterArgs) (string, error) {

	fd := int32(args[0])
	cmd := int32(args[1])

	cmdstr := fcntlcmds[cmd]

	arg3 := ""

	switch args[1] {

	case C.F_SETLK, C.F_GETLK, C.F_SETLKW:
		var flock syscall.Flock_t
		buf, err := readBytesArg(pid, int(unsafe.Sizeof(flock)), uintptr(args[2]))
		if err != nil {
			return "", err
		}
		b := bytes.NewBuffer(buf)
		binary.Read(b, binary.LittleEndian, &flock)
		arg3 = render_Flock_t(flock)
	default:
		arg3 = fmt.Sprintf("%x", args[2])
	}

	callrep := fmt.Sprintf("fcntl(%d,%s,%s)", fd, cmdstr, arg3)

	return callrep, nil

}

func render_Flock_t(flock syscall.Flock_t) string {
	typestr := fcntllock[flock.Type]
	whence := whences[flock.Whence]

	return fmt.Sprintf("{type=%s, whence=%s, start=%v, len=%v}", typestr, whence, flock.Start, flock.Len)
}
