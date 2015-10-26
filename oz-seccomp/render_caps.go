package seccomp

import (
	"fmt"
)

// #include "linux/capability.h"
import "C"

var flags = map[uint]string{
	C._LINUX_CAPABILITY_VERSION_1: "_LINUX_CAPABILITY_VERSION_1",
	C._LINUX_CAPABILITY_VERSION_2: "_LINUX_CAPABILITY_VERSION_2",
	C._LINUX_CAPABILITY_VERSION_3: "_LINUX_CAPABILITY_VERSION_3",
}

func render_capget(pid int, args RegisterArgs) (string, error) {

	buf, err := readBytesArg(pid, 8, uintptr(args[0]))

	if err != nil {
		return "", err
	}

	/*

		typedef struct __user_cap_header_struct {
			        __u32 version;
				        int pid;
				} *cap_user_header_t;


	*/

	version := bytestoint32(buf[:4])
	cpid := bytestoint32(buf[4:])
	addr := ptrtostrornull(uintptr(args[1]))
	callrep := fmt.Sprintf("capget({version=%s,pid=%d},%s)", flags[uint(version)], cpid, addr)
	return callrep, nil
}
