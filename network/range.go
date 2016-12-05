package network

import (
	"github.com/j-keck/arping"
	"math/rand"
	"net"
	"os"
	"time"
)

type pinger interface {
	ping(dst net.IP) bool
}

type arpPinger struct {
	iface string
}

func (ap *arpPinger) ping(dst net.IP) bool {
	_, _, err := arping.PingOverIfaceByName(dst, ap.iface)
	return err == nil
}

func init() {
	arping.SetTimeout(time.Millisecond * 150)
	rand.Seed(time.Now().UnixNano() ^ int64(os.Getpid()))
}

// IPRange represents a subnet range from which individual IP addresses can be allocated
type IPRange struct {
	*net.IPNet
	inUse  []bool
	first  uint32 // first usable IP address in the range
	size   int    // number of usable addresses in the range
	pinger pinger
}

func newIPRange(ipnet *net.IPNet, iface string) *IPRange {
	return newIPRangeWithPinger(ipnet, &arpPinger{iface})
}

func newIPRangeWithPinger(ipnet *net.IPNet, pinger pinger) *IPRange {
	sz := networkSize(ipnet.Mask)
	inUse := make([]bool, sz)
	inUse[0] = true

	return &IPRange{
		IPNet:  ipnet,
		first:  firstIP(ipnet),
		inUse:  inUse,
		size:   sz,
		pinger: pinger,
	}
}

func (ipr *IPRange) FirstIP() net.IP {
	return ipr.usableAt(0)
}

func (ipr *IPRange) FreshIP() net.IP {
	if ip := ipr.randomIP(); ip != nil {
		return ip
	}
	return ipr.scanIP()
}

func (ipr *IPRange) randomIP() net.IP {
	for i := 0; i < ozMaxRandTries; i++ {
		offset := 1 + rand.Intn(ipr.size-1)
		if ip := ipr.availableAt(offset); ip != nil {
			return ip
		}
	}
	return nil
}

func (ipr *IPRange) scanIP() net.IP {
	for i := 1; i < ipr.size; i++ {
		if ip := ipr.availableAt(i); ip != nil {
			return ip
		}
	}
	return nil
}

func networkSize(mask net.IPMask) int {
	bits, _ := mask.Size()
	switch bits {
	case 32:
		return 1
	case 31:
		return 2
	default:
		return pow2(32-bits) - 2
	}
}

func firstIP(ipnet *net.IPNet) uint32 {
	bits, _ := ipnet.Mask.Size()
	n := toUint32(ipnet.IP)
	if bits < 31 {
		n += 1
	}
	return n
}

func (ipr *IPRange) availableAt(offset int) net.IP {
	if offset < 0 || offset >= ipr.size {
		return nil
	}
	if ipr.inUse[offset] {
		return nil
	}
	ip := ipr.usableAt(offset)
	if ipr.pinger.ping(ip) {
		ipr.inUse[offset] = true
		return nil
	}
	ipr.inUse[offset] = true
	return ip
}

// usableAt returns an IPv4 address from this IPRange at offset
// from the first usable address in the range.  If offset is
// negative or greater or equal to number of usable addresses
// this function will panic.
func (ipr *IPRange) usableAt(offset int) net.IP {
	if offset < 0 || offset >= ipr.size {
		panic("offset has illegal value")
	}
	return toIP(ipr.first + uint32(offset))
}

// pow2 returns 2 raised to the power exp for exponents
// between 0 and 31 inclusive.
func pow2(exp int) int {
	if exp < 0 || exp > 31 {
		panic("exponent must be between 0 and 31")
	}
	n := 1
	for i := 0; i < exp; i++ {
		n *= 2
	}
	return n
}

// toUint32 converts the net.IP address ip into uint32 value
// in network byte order
func toUint32(ip net.IP) uint32 {
	n := uint32(0)
	for _, b := range ip.To4() {
		n <<= 8
		n |= uint32(b)
	}
	return n
}

// toIP converts a uint32 value representing an IPv4 address into
// a 4 byte net.IP value
func toIP(n uint32) net.IP {
	return net.IPv4(
		byte(n>>24),
		byte(n>>16),
		byte(n>>8),
		byte(n),
	).To4()
}
