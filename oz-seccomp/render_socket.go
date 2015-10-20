package seccomp

import (
	"fmt"
	"syscall"
)

var protocols = map[uint]string{
	syscall.IPPROTO_IP:   "IPPROTO_IP",
	syscall.IPPROTO_ICMP: "IPPROTO_ICMP",
	syscall.IPPROTO_IGMP: "IPPROTO_IGMP",
	syscall.IPPROTO_IPIP: "IPPROTO_IPIP",
	syscall.IPPROTO_TCP:  "IPPROTO_TCP",
	syscall.IPPROTO_EGP:  "IPPROTO_EGP",
	syscall.IPPROTO_PUP:  "IPPROTO_PUP",
	syscall.IPPROTO_UDP:  "IPPROTO_UDP",
	syscall.IPPROTO_IDP:  "IPPROTO_IDP",
	syscall.IPPROTO_TP:   "IPPROTO_TP",
	syscall.IPPROTO_DCCP: "IPPROTO_DCCP",
	syscall.IPPROTO_IPV6: "IPPROTO_IPV6",
	syscall.IPPROTO_RSVP: "IPPROTO_RSVP",
	syscall.IPPROTO_GRE:  "IPPROTO_GRE",
	syscall.IPPROTO_ESP:  "IPPROTO_ESP",
	syscall.IPPROTO_AH:   "IPPROTO_AH",
	syscall.IPPROTO_MTP:  "IPPROTO_MTP",
	//syscall.IPPROTO_BEETPH: "IPPROTO_BEETPH",
	syscall.IPPROTO_ENCAP:   "IPPROTO_ENCAP",
	syscall.IPPROTO_PIM:     "IPPROTO_PIM",
	syscall.IPPROTO_COMP:    "IPPROTO_COMP",
	syscall.IPPROTO_SCTP:    "IPPROTO_SCTP",
	syscall.IPPROTO_UDPLITE: "IPPROTO_UDPLITE",
	syscall.IPPROTO_RAW:     "IPPROTO_RAW",
}

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

	domain := ""
	socktype := ""

	prot := fmt.Sprintf("%d", uint(args[2]))

	if uint(args[0]) == syscall.AF_INET || uint(args[0]) == syscall.AF_INET6 {
		if _, y := protocols[uint(args[2])]; y {
			prot = protocols[uint(args[2])]
		}
	}
	if _, y := domainflags[uint(args[0])]; y {
		domain = domainflags[uint(args[0])]
	} else {
		domain = fmt.Sprintf("%d", args[0])
	}

	socktype = socktypes[uint(args[1]&0xFF)]
	socktypestr := socktype

	tmp := renderFlags(socktypeflags, uint(args[1]))

	if tmp != "" {
		socktypestr += "|"
	}

	socktypestr += tmp

	callrep := fmt.Sprintf("socket(%s, %s, %s)", domain, socktypestr, prot)
	return callrep, nil
}

	
