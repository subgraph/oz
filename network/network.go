package network

import (
	//Builtin
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/op/go-logging"
	"github.com/milosgajdos83/tenus"
)

const (
	ozDefaultInterfaceBridgeBase = "oz"
	ozDefaultInterfaceBridge     = ozDefaultInterfaceBridgeBase + "0"
	ozDefaultInterfacePrefix     = "veth"
	ozDefaultInterfaceInternal   = "eth0"
	ozMaxRandTries               = 3
)

type NetType string

const(
	TYPE_HOST NetType   = "host"
	TYPE_EMPTY NetType  = "empty"
	TYPE_BRIDGE NetType = "bridge"
)

type HostNetwork struct {
	// Bridge interface
	Interface tenus.Bridger
	// Gateway ip (bridge ip)
	Gateway net.IP
	// Gateway ip (bridge ip)
	GatewayNet *net.IPNet
	// Bridge netmask
	Netmask net.IP
	// Broadcast ip
	Broadcast net.IP
	// IP class (ie: /24)
	Class string
	// Minimum longip available ip
	Min uint64
	// Maximum longip available ip
	Max uint64
	// Bridge interface MAC Address
	BridgeMAC string
	// The type of network configuration
	Nettype NetType
}

type SandboxNetwork struct {
	// veth interface is present
	Interface tenus.Linker
	// Name of the veth in the host
	VethHost string
	// Temporary name of the guest' veth in the host
	VethGuest string
	// Guest ip address
	Ip string
	// Gateway ip (bridge ip)
	Gateway net.IP
	// IP class (ie: /24)
	Class string
	// The type of network configuration
	Nettype NetType
	// Host side virtual interface
	Veth tenus.Vether
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

// Print status of the network interfaces
func NetPrint(log *logging.Logger) {
	strLine := ""
	ifs, _ := net.Interfaces()

	strHeader := fmt.Sprintf("%-15.15s%-30.30s%-16.16s%-6.6s", "Interface", "IP", "Mask", "Status")
	strHr := ""
	ii := len(strHeader)
	for i := 0; i < ii; i++ {
		strHr += "-"
	}

	log.Info(strHr)

	log.Info(strHeader)

	for _, netif := range ifs {
		if strings.HasPrefix(netif.Name, ozDefaultInterfacePrefix) {
			continue
		}
		addrs, _ := netif.Addrs()

		strLine = fmt.Sprintf("%-15.14s", netif.Name)

		if len(addrs) > 0 {
			strLine += fmt.Sprintf("%-30.30s", addrs[0])

			bIP, brIP, _ := net.ParseCIDR(addrs[0].String())
			if bIP.To4() != nil {
				bMask := []byte(brIP.Mask)
				strLine += fmt.Sprintf("%-16.16s", net.IPv4(bMask[0], bMask[1], bMask[2], bMask[3]).String())
			} else {
				strLine += fmt.Sprintf("%-16.16s", "")
			}
		} else {
			strLine += fmt.Sprintf("%-30.30s%-16.16s", "", "")
		}

		if netif.Flags&net.FlagUp == 1 {
			strLine += fmt.Sprintf("%-6.6s", "up")
		} else {
			strLine += fmt.Sprintf("%-6.6s", "down")
		}

		if len(addrs) > 1 {
			strLine += fmt.Sprintf("")

			for _, addr := range addrs[1:] {
				strLine += fmt.Sprintf("%-15.15s%-30.30s", "", addr)

				bIP, brIP, _ := net.ParseCIDR(addr.String())

				if bIP.To4() != nil {
					bMask := []byte(brIP.Mask)
					strLine += fmt.Sprintf("%-20.20s", net.IPv4(bMask[0], bMask[1], bMask[2], bMask[3]).String())
				} else {
					strLine += fmt.Sprintf("%-16.16s", "")
				}
			}
		}

		strLine += fmt.Sprintf("\n")

		log.Info(strLine)
	}

	log.Info(strHr)

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
