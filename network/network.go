package network

import (
	//Builtin
	"net"
	"strings"
	"strconv"
)

const (
	ozDefaultInterfaceBridgeBase = "oz"
	ozDefaultInterfaceBridge     = ozDefaultInterfaceBridgeBase + "0"
	ozDefaultInterfacePrefix     = "veth"
	ozDefaultInterfaceInternal   = "eth0"
	ozMaxRandTries               = 3
)

type HostNetwork struct {
	// Host bridge IP address
	hostip net.IP
	// Gateway ip (bridge ip)
	gateway net.IP
	// Bridge netmask
	netmask net.IP
	// Broadcast ip
	broadcast net.IP
	// IP class (ie: /24)
	class string
	// Minimum longip available ip
	min uint64
	// Maximum longip available ip
	max uint64

}

type SandboxNetwork struct {
	// Name of the veth in the host
	vethHost string
	// Temporary name of the guest' veth in the host
	vethGuest string
	// Guest ip address
	ip string
	
	host *HostNetwork
}

var privateNetworkRanges []string

func init() {
	privateNetworkRanges = []string{
		// RFC1918 Private ranges
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		// Documentation / Testnet 2
		"192.51.100.0/24",
		// Documentation/ Testnet
		"192.0.2.0/24",
		// Inter-network communication
		"192.18.0.0/15",
		// Documentation / Testnet 3
		"203.0.113.0/24",
		// Carrier grade NAT
		"100.64.0.0/10",
	}
}


// Convert longip to net.IP
func inet_ntoa(ipnr uint64) net.IP {
	var bytes [4]byte
	bytes[0] = byte(ipnr & 0xFF)
	bytes[1] = byte((ipnr >> 8) & 0xFF)
	bytes[2] = byte((ipnr >> 16) & 0xFF)
	bytes[3] = byte((ipnr >> 24) & 0xFF)

	return net.IPv4(bytes[3], bytes[2], bytes[1], bytes[0])
}

// Convert net.IP to longip
func inet_aton(ipnr net.IP) uint64 {
	bits := strings.Split(ipnr.String(), ".")

	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])

	var sum uint64

	sum += uint64(b0) << 24
	sum += uint64(b1) << 16
	sum += uint64(b2) << 8
	sum += uint64(b3)

	return sum
}

func net_getbroadcast(bIP net.IP, ipMask net.IPMask) net.IP {
	bMask := []byte(ipMask)
	byteIP := bIP.To4()

	return net.IPv4(
		byteIP[0]|(bMask[0]^0xFF),
		byteIP[1]|(bMask[1]^0xFF),
		byteIP[2]|(bMask[2]^0xFF),
		byteIP[3]|(bMask[3]^0xFF))

}
