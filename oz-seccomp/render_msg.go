package seccomp

import (
	"fmt"
	"syscall"
)

var msgflags = map[uint]string{
	syscall.MSG_OOB:          "MSG_OOB",
	syscall.MSG_PEEK:         "MSG_PEEK",
	syscall.MSG_DONTROUTE:    "MSG_DONTROUTE",
	syscall.MSG_CTRUNC:       "MSG_CTRUNC",
	syscall.MSG_PROXY:        "MSG_PROXY",
	syscall.MSG_TRUNC:        "MSG_TRUNC",
	syscall.MSG_DONTWAIT:     "MSG_DONTWAIT",
	syscall.MSG_EOR:          "MSG_EOR",
	syscall.MSG_WAITALL:      "MSG_WAITALL",
	syscall.MSG_FIN:          "MSG_FIN",
	syscall.MSG_SYN:          "MSG_SYN",
	syscall.MSG_CONFIRM:      "MSG_CONFIRM",
	syscall.MSG_RST:          "MSG_RST",
	syscall.MSG_ERRQUEUE:     "MSG_ERRQUEUE",
	syscall.MSG_NOSIGNAL:     "MSG_NOSIGNAL",
	syscall.MSG_MORE:         "MSG_MORE",
	syscall.MSG_WAITFORONE:   "MSG_WAITFORONE",
	syscall.MSG_FASTOPEN:     "MSG_FASTOPEN",
	syscall.MSG_CMSG_CLOEXEC: "MSG_CMSG_CLOEXEC",
}

func render_recvmsg(pid int, args RegisterArgs) (string, error) {

	flagstr := renderFlags(msgflags, uint(args[2]))

	if flagstr == "" {
		flagstr = "0"
	}

	callrep := fmt.Sprintf("recvmsg(%d, 0x%X, %s)", args[0], args[1], flagstr)
	return callrep, nil
}
