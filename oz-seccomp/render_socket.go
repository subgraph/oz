package seccomp

import (
	"fmt"
	"syscall"
)

// #include "linux/netlink.h"
import "C"

const (
	SOL_NETLINK  = 270 // missing from syscall
	SO_REUSEPORT = 15  // missing from syscall
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

var sockoptlevels = map[uint]string{
	syscall.SOL_AAL:    "SOL_AAL",
	syscall.SOL_DECNET: "SOL_DECNET",
	syscall.SOL_ATM:    "SOL_ATM",
	syscall.SOL_ICMPV6: "SOL_ICMPV6",
	syscall.SOL_IRDA:   "SOL_IRDA",
	syscall.SOL_IP:     "SOL_IP",
	syscall.SOL_IPV6:   "SOL_IPV6",
	syscall.SOL_PACKET: "SOL_PACKET",
	syscall.SOL_RAW:    "SOL_RAW",
	syscall.SOL_SOCKET: "SOL_SOCKET",
	syscall.SOL_TCP:    "SOL_TCP",
	syscall.SOL_X25:    "SOL_X25",
	SOL_NETLINK:        "SOL_NETLINK",
}

var tcpopts = map[uint]string{
	syscall.TCP_CONGESTION:   "TCP_CONGESTION",
	syscall.TCP_CORK:         "TCP_CORK",
	syscall.TCP_DEFER_ACCEPT: "TCP_DEFER_ACCEPT",
	syscall.TCP_INFO:         "TCP_INFO",
	syscall.TCP_KEEPCNT:      "TCP_KEEPCNT",
	syscall.TCP_KEEPIDLE:     "TCP_KEEPIDLE",
	syscall.TCP_KEEPINTVL:    "TCP_KEEPINTVL",
	syscall.TCP_LINGER2:      "TCP_LINGER2",
	syscall.TCP_MAXSEG:       "TCP_MAXSEG",
	syscall.TCP_MAXWIN:       "TCP_MAXWIN",
	syscall.TCP_MAX_WINSHIFT: "TCP_MAX_WINSHIFT", //0xe
	//	syscall.TCP_MD5SIG: "TCP_MD5SIG", // 0xe
	syscall.TCP_MSS:              "TCP_MSS",
	syscall.TCP_MD5SIG_MAXKEYLEN: "TCP_MD5SIG_MAXKEYLEN",
	syscall.TCP_NODELAY:          "TCP_NODELAY",
	syscall.TCP_QUICKACK:         "TCP_QUICKACK",
	syscall.TCP_SYNCNT:           "SYP_SYNCNT",
	syscall.TCP_WINDOW_CLAMP:     "TCP_WINDOW_CLAMP",
}

var netlinkopts = map[uint]string{
	C.NETLINK_ADD_MEMBERSHIP:   "NETLINK_ADD_MEMBERSHIP",
	C.NETLINK_DROP_MEMBERSHIP:  "NETLINK_DROP_MEMBERSHIP",
	C.NETLINK_PKTINFO:          "NETLINK_PKTINFO",
	C.NETLINK_BROADCAST_ERROR:  "NETLINK_BROADCAST_ERROR",
	C.NETLINK_NO_ENOBUFS:       "NETLINK_NO_ENOBUFS",
	C.NETLINK_RX_RING:          "NETLINK_RX_RING",
	C.NETLINK_TX_RING:          "NETLINK_TX_RING",
	C.NETLINK_LISTEN_ALL_NSID:  "NETLINK_LISTEN_ALL_NSID",
	C.NETLINK_LIST_MEMBERSHIPS: "NETLINK_LIST_MEMBERSHIPS",
	C.NETLINK_CAP_ACK:          "NETLINK_CAP_ACK",
}

var sockopts = map[uint]string{

	syscall.SO_ACCEPTCONN:                    "SO_ACCEPTCONN",
	syscall.SO_ATTACH_FILTER:                 "SO_ATTACH_FILTER",
	syscall.SO_BINDTODEVICE:                  "BINDTODEVICE",
	syscall.SO_BROADCAST:                     "SO_BROADCAST",
	syscall.SO_BSDCOMPAT:                     "SO_BSDCOMPAT",
	syscall.SO_DEBUG:                         "SO_DEBUG",
	syscall.SO_DETACH_FILTER:                 "SO_DETACH_FILTER",
	syscall.SO_DOMAIN:                        "SO_DOMAIN",
	syscall.SO_DONTROUTE:                     "SO_DONTROUTE",
	syscall.SO_ERROR:                         "SO_ERROR",
	syscall.SO_KEEPALIVE:                     "SO_KEEPALIVE",
	syscall.SO_LINGER:                        "SO_LINGER",
	syscall.SO_MARK:                          "SO_MARK",
	syscall.SO_NO_CHECK:                      "SO_NO_CHECK",
	syscall.SO_OOBINLINE:                     "SO_OOBINLINE",
	syscall.SO_PASSCRED:                      "SO_PASSCRED",
	syscall.SO_PASSSEC:                       "SO_PASSSEC",
	syscall.SO_PEERCRED:                      "SO_PEERCRED",
	syscall.SO_PEERSEC:                       "SO_PEERSEC",
	syscall.SO_PRIORITY:                      "SO_PRIORITY",
	syscall.SO_PROTOCOL:                      "SO_PROTOCOL",
	syscall.SO_RCVBUF:                        "SO_RCVBUF",
	syscall.SO_RCVBUFFORCE:                   "SO_RCVBUFFORCE",
	syscall.SO_RCVLOWAT:                      "SO_RCVLOWAT",
	syscall.SO_RCVTIMEO:                      "SO_RCVTIMEO",
	syscall.SO_REUSEADDR:                     "SO_REUSEADDR",
	SO_REUSEPORT:                             "SO_REUSEPORT",
	syscall.SO_RXQ_OVFL:                      "SO_RXQ_OVFL",
	syscall.SO_SECURITY_AUTHENTICATION:       "SO_SECURITY_AUTHENTICATION",
	syscall.SO_SECURITY_ENCRYPTION_NETWORK:   "SO_SECURITY_ENCRYPTION_NETWORK",
	syscall.SO_SECURITY_ENCRYPTION_TRANSPORT: "SO_SECURITY_ENCRYPTION_TRANSPORT",
	syscall.SO_SNDBUF:                        "SO_SNDBUF",
	syscall.SO_SNDBUFFORCE:                   "SO_SNDBUFFORCE",
	syscall.SO_SNDLOWAT:                      "SO_SNDLOWAT",
	syscall.SO_SNDTIMEO:                      "SO_SNDTIMEO",
	syscall.SO_TIMESTAMP:                     "SO_TIMESTAMP",
	syscall.SO_TIMESTAMPING:                  "SO_TIMESTAMPING",
	syscall.SO_TIMESTAMPNS:                   "SO_TIMESTAMPNS",
	syscall.SO_TYPE:                          "SO_TYPE",
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

func render_connect(pid int, args RegisterArgs) (string, error) {

	// TODO Watch the length supplied by the process

	buf, err := readBytesArg(pid, int(args[2]), uintptr(args[1]))
	if err != nil {
		return "", err
	}

	fam := bytestoint16(buf[:2])
	addrstruct := ""
	addrstruct = render_inetaddr(buf)
	switch fam {
	case syscall.AF_INET:
		addrstruct = render_inetaddr(buf)
	case syscall.AF_UNIX:
		addrstruct = render_unixaddr(buf)
	}

	callrep := fmt.Sprintf("connect(%d, %s, %d)", args[0], addrstruct, args[2])

	return callrep, nil

}

func render_setsockopt(pid int, args RegisterArgs) (string, error) {

	var opt string
	level := sockoptlevels[uint(args[1])]

	switch uint(args[1]) {
	case syscall.SOL_SOCKET:
		opt = sockopts[uint(args[2])]
	case syscall.SOL_TCP:
		opt = tcpopts[uint(args[2])]
	case SOL_NETLINK:
		opt = netlinkopts[uint(args[2])]
	default:
		opt = fmt.Sprintf("%d", args[2])
	}

	callrep := fmt.Sprintf("setsockopt(%d, %s, %s, 0x%X, %d)", args[0], level, opt, args[3], args[4])

	return callrep, nil
}
