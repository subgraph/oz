package openvpn

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"os/exec"
)

func Up() {
	var n, bn net.IPNet
	var i net.IP
	var s string

	/* Supplied by OpenVPN Provider */

	ozdebug := os.Getenv("oz_debug")

	ipgwstr := os.Getenv("route_network_1")
	if ipgwstr == "" {
		ipgwstr = os.Getenv("route_vpn_gateway")
	}
	ifrstr := os.Getenv("ifconfig_remote")
	iflstr := os.Getenv("ifconfig_local")
	dev := os.Getenv("dev")

	/* Supplied by Oz */

	bridgeaddr := os.Getenv("bridge_addr")
	bridgedev := os.Getenv("bridge_dev")
	table := os.Getenv("routing_table")

	/* Need to decide how to exit if params from
	   OpenVPN server missing or invalid
	*/

	var mask net.IPMask
	i = net.ParseIP(ipgwstr)

	ifnetmask := os.Getenv("ifconfig_netmask")
	if ifnetmask != "" {
		mask = ParseIPv4Mask(ifnetmask)
	} else {
		mask = net.CIDRMask(24, 32)
	}
	i = i.Mask(mask)
	n.Mask = mask
	n.IP = i

	/* Oz bridge is always /24 */

	bi := net.ParseIP(bridgeaddr)
	bmask := net.CIDRMask(24, 32)
	bi = bi.Mask(bmask)
	bn.Mask = bmask
	bn.IP = bi

	if ozdebug != "" {

		ff := os.Environ()
		for i := range ff {
			s += ff[i]
			s += "\n"
		}
	}

	s += fmt.Sprintf("echo Adding to table %s:\n", table)

	//	cmd := exec.Command("/bin/ip", "route", "add", n_String(), dev, table)
	//	cmd.Run()

	s += fmt.Sprintf("/bin/ip route add %s dev %s scope link table %s\n", n.String(), dev, table)

	cmd := exec.Command("/bin/ip", "route", "add", n.String(), "dev", dev, "scope", "link", "table", table)
	cmd.Run()

	s += fmt.Sprintf("/bin/ip route add %s dev %s proto kernel scope link src %s table %s\n", ifrstr, dev, iflstr, table)

	cmd = exec.Command("/bin/ip", "route", "add", ifrstr, "dev", dev, "proto", "kernel", "scope", "link", "src", iflstr, "table", table)
	cmd.Run()

	s += fmt.Sprintf("/bin/ip route add %s dev %s proto kernel scope link src %s table %s\n", bn.String(), bridgedev, bridgeaddr, table)

	cmd = exec.Command("/bin/ip", "route", "add", bn.String(), "dev", bridgedev, "proto", "kernel", "scope", "link", "src", bridgeaddr, "table", table)
	cmd.Run()

	s += fmt.Sprintf("/bin/ip route add default via %s dev %s table %s\n", ipgwstr, dev, table)

	cmd = exec.Command("/bin/ip", "route", "add", "default", "via", ipgwstr, "dev", dev, "table", table)
	cmd.Run()

	/* Policy rules */

	s += fmt.Sprintf("echo Adding policy rules:\n")
	s += fmt.Sprintf("ip rule add from all to %s lookup %s\n", iflstr, table)

	cmd = exec.Command("/bin/ip", "rule", "add", "from", "all", "to", iflstr, "lookup", table)
	cmd.Run()
	s += fmt.Sprintf("ip rule add from %s lookup %s\n", iflstr, table)

	cmd = exec.Command("/bin/ip", "rule", "add", "from", iflstr, "lookup", table)
	cmd.Run()

	s += fmt.Sprintf("ip rule add from %s lookup %s\n", bn.String(), table)

	cmd = exec.Command("/bin/ip", "rule", "add", "from", bn.String(), "lookup", table)
	cmd.Run()

	s += fmt.Sprintf("ip rule add from all to %s lookup %s\n", bn.String(), table)

	cmd = exec.Command("/bin/ip", "rule", "add", "from", "all", "to", bn.String(), "lookup", table)
	if ozdebug != "" {
		fmt.Fprintf(os.Stderr, s)
	}

	cmd.Run()
}

/* Below ripped out of github.com/spf13/pflag, did the trick, thanks! */

func ParseIPv4Mask(s string) net.IPMask {
	mask := net.ParseIP(s)
	if mask == nil {
		if len(s) != 8 {
			return nil
		}
		// net.IPMask.String() actually outputs things like ffffff00
		// so write a horrible parser for that as well  :-(
		m := []int{}
		for i := 0; i < 4; i++ {
			b := "0x" + s[2*i:2*i+2]
			d, err := strconv.ParseInt(b, 0, 0)
			if err != nil {
				return nil
			}
			m = append(m, int(d))
		}
		s := fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
		mask = net.ParseIP(s)
		if mask == nil {
			return nil
		}
	}
	return net.IPv4Mask(mask[12], mask[13], mask[14], mask[15])
}
