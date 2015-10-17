package seccomp

import (
	"fmt"
	"syscall"
)

var domainflags = map[uint]string{
	syscall.AF_UNIX:      "AF_UNIX",
	syscall.AF_INET:      "AF_INET",
	syscall.AF_INET6:     "AF_INET6",
	syscall.AF_IPX:       "AF_IPX",
	syscall.AF_NETLINK:   "AF_NETLINK",
	syscall.AF_X25:       "AF_X25",
	syscall.AF_AX25:      "AF_AX25",
	syscall.AF_ATMPVC:    "AF_ATMPVC",
	syscall.AF_APPLETALK: "AF_APPLETALK",
	syscall.AF_PACKET:    "AF_PACKET",
	//	syscall.ALG: "AF_ALG",
}

var socktypes = map[uint]string{
	syscall.SOCK_STREAM:    "SOCK_STREAM",
	syscall.SOCK_DGRAM:     "SOCK_DGRAM",
	syscall.SOCK_SEQPACKET: "SOCK_SEQPACKET",
	syscall.SOCK_RAW:       "SOCK_RAW",
	syscall.SOCK_RDM:       "SOCK_RDM",
	syscall.SOCK_PACKET:    "SOCK_PACKET",
}

var socktypeflags = map[uint]string{
	syscall.SOCK_NONBLOCK: "SOCK_NONBLOCK",
	syscall.SOCK_CLOEXEC:  "SOCK_CLOEXEC",
}

func render_socket(pid int, args RegisterArgs) (string, error) {

	domain := domainflags[uint(args[0])]
	socktype := socktypes[uint(args[1]&0xFF)]

	socktypestr := socktype
	tmp := renderFlags(socktypeflags, uint(args[1]))

	if tmp != "" {
		socktypestr += "|"
	}

	socktypestr += tmp

	callrep := fmt.Sprintf("socket(%s, %s, %d)", domain, socktypestr, args[2])
	return callrep, nil
}
