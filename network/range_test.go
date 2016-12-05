package network

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"
)

func assertPanic(t *testing.T, msg string, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("function did not panic() as expected: %s", msg)
		}
	}()
	f()
}

type testPinger struct {
	used map[uint32]bool
}

func (tp *testPinger) addUsed(ip net.IP) {
	tp.used[toUint32(ip)] = true
}

func (tp *testPinger) ping(dst net.IP) bool {
	return tp.used[toUint32(dst)]
}

func newTestRange(cidr string, used ...string) *IPRange {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic("failed to parse CIDR: " + cidr)
	}
	tp := &testPinger{used: make(map[uint32]bool)}
	for _, s := range used {
		ip := net.ParseIP(s)
		if ip == nil {
			panic("could not parse ip: " + s)
		}
		tp.addUsed(ip)
	}
	return newIPRangeWithPinger(ipnet, tp)
}

type ipgen func() net.IP

func runRangeTest(t *testing.T, suffix []byte, g ipgen) {
	for _, b := range suffix {
		if ip := g(); ip == nil || ip[3] != b {
			got := ip.String()
			ip[3] = b
			t.Errorf("expecting %v, got %s", ip, got)
		}
	}
}

func TestFreshIP(t *testing.T) {
	runRangeTest(t, []byte{2, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14},
		newTestRange("192.168.1.0/28", "192.168.1.3", "192.168.1.8").scanIP)

	rand.Seed(123)
	runRangeTest(t, []byte{6, 4, 13, 5, 2, 12, 9, 7, 11, 10, 14},
		newTestRange("192.168.1.0/28", "192.168.1.3", "192.168.1.8").FreshIP)
}

var firstIPTestData = []struct {
	cidr  string
	first net.IP
}{
	{"192.168.1.0/32", net.IPv4(192, 168, 1, 0)},
	{"192.168.1.0/31", net.IPv4(192, 168, 1, 0)},
	{"192.168.1.0/30", net.IPv4(192, 168, 1, 1)},
	{"192.168.1.0/29", net.IPv4(192, 168, 1, 1)},
}

func TestFirstIP(t *testing.T) {
	for _, tst := range firstIPTestData {
		ipr := newTestRange(tst.cidr)
		first := ipr.FirstIP()
		if !first.Equal(tst.first) {
			t.Errorf("FirstIP(%s) = %v, expected %v", tst.cidr, first, tst.first)
		}
	}
}

func TestPow2(t *testing.T) {
	for i, d := range []int{1, 2, 4, 8, 16, 32} {
		if pow2(i) != d {
			t.Errorf("pow2(%d) != %d", i, d)
		}
	}
	if pow2(31) != 2147483648 {
		t.Errorf("pow2(31) != 2147483648")
	}

	for _, bad := range []int{-1, -100, 32, 100} {
		msg := fmt.Sprintf("pow2(%d)", bad)
		assertPanic(t, msg, func() {
			pow2(bad)
		})
	}
}

var uint32Tests = []struct {
	ip net.IP
	n  uint32
}{
	{net.IPv4(1, 2, 3, 4), 0x01020304},
	{net.IPv4(0, 0, 0, 0), 0x00000000},
	{net.IPv4(255, 255, 255, 255), 0xFFFFFFFF},
}

func TestUint32(t *testing.T) {
	for _, tst := range uint32Tests {
		if n := toUint32(tst.ip); n != tst.n {
			t.Errorf("toUint32(%v) = %x, want %x", tst.ip, n, tst.n)
		}
	}
}

var toIPTests = []struct {
	n  uint32
	ip net.IP
}{
	{0x01020304, net.IPv4(1, 2, 3, 4).To4()},
	{0x00000000, net.IPv4(0, 0, 0, 0).To4()},
	{0xFFFFFFFF, net.IPv4(255, 255, 255, 255).To4()},
}

func TestToIP(t *testing.T) {
	for _, tst := range toIPTests {
		if ip := toIP(tst.n); !reflect.DeepEqual(ip, tst.ip) {
			t.Errorf("toIP(%x) = %v, want %v", tst.n, ip, tst.ip)
		}
	}
}
