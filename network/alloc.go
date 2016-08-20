package network

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"net"
	"strings"
)

type subnetAllocator struct {
	baseNet    *net.IPNet
	nextSubnet int
	log        *logging.Logger
}

func (sa *subnetAllocator) allocateRange(iface string) (*IPRange, error) {
	n, err := sa.allocate()
	if err != nil {
		return nil, err
	}
	sa.log.Infof("Allocating new subnet range (%v) for interface '%s'", n, iface)
	return newIPRange(n, iface), nil
}

func (sa *subnetAllocator) allocate() (*net.IPNet, error) {
	if sa.nextSubnet > 255 {
		return nil, fmt.Errorf("Cannot allocate any more subnets from %v", sa.baseNet)
	}

	ip4 := sa.baseNet.IP.To4()

	sub := &net.IPNet{
		IP:   net.IPv4(ip4[0], ip4[1], byte(sa.nextSubnet), 0).To4(),
		Mask: net.IPv4Mask(255, 255, 255, 0)}
	sa.nextSubnet += 1
	return sub, nil
}

func (sa *subnetAllocator) needsReconfigure() bool {
	return overlapsAny(sa.baseNet, getLocalNetworks())
}

func newAllocator(log *logging.Logger) (*subnetAllocator, error) {
	base := findBaseNet(getLocalNetworks())
	if base == nil {
		return nil, errors.New("Unable to find unused /16 network")
	}
	log.Infof("Subnet allocator created with base network: %v", base)
	return &subnetAllocator{
		baseNet:    base,
		nextSubnet: 1,
		log:        log,
	}, nil
}

func findBaseNet(localNets []*net.IPNet) *net.IPNet {
	for _, pn := range privateNetworks {
		if !overlapsAny(pn, localNets) {
			return &net.IPNet{
				IP:   pn.IP.To4(),
				Mask: net.IPv4Mask(255, 255, 0, 0),
			}
		}
	}
	return nil
}

func overlapsAny(a *net.IPNet, networks []*net.IPNet) bool {
	for _, n := range networks {
		if overlaps(a, n) {
			return true
		}
	}
	return false
}

func overlaps(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

func getInterfaces() []net.Interface {
	ifs, err := net.Interfaces()
	if err != nil {
		panic(fmt.Sprintf("Error retrieving list of system interfaces: %v", err))
	}
	return filterInterfaces(ifs)
}

func getLocalNetworks() []*net.IPNet {
	var networks []*net.IPNet
	for _, ifa := range getInterfaces() {
		as, err := ifa.Addrs()
		if err != nil {
			panic(fmt.Sprintf("Error reading addresses from interface %s: %v", ifa.Name, err))
		}
		for _, a := range as {
			if ipn := addressToIPv4Net(a); ipn != nil {
				networks = append(networks, ipn)
			}
		}
	}
	return networks
}

func addressToIPv4Net(a net.Addr) *net.IPNet {
	_, ipnet, err := net.ParseCIDR(a.String())
	if err != nil {

	}
	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return nil
	}
	ipnet.IP = ip4
	return ipnet
}

func filterInterfaces(ifs []net.Interface) []net.Interface {
	var result []net.Interface
	for _, ifa := range ifs {
		name := ifa.Name
		if !strings.HasPrefix(name, "lo") &&
			!strings.HasPrefix(name, ozDefaultInterfaceBridgeBase) &&
			!strings.HasPrefix(name, ozDefaultInterfacePrefix) {
			result = append(result, ifa)
		}
	}
	return result
}

var privateNetworks = parseRanges(
	// RFC1918 Private ranges
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	// Inter-network communication
	"192.18.0.0/15",
	// Carrier grade NAT
	"100.64.0.0/10",
)

func parseRanges(rs ...string) []*net.IPNet {
	var ranges []*net.IPNet
	for _, s := range rs {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse private range (%s): %v", s, err))
		}
		ensureSize(n)
		ranges = append(ranges, n)
	}
	return ranges
}

func ensureSize(n *net.IPNet) {
	ones, bits := n.Mask.Size()
	if ones > 16 || bits != 32 {
		panic(fmt.Sprintf("Network (%s) is not an appropriate size", n))
	}
}
