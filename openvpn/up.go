package openvpn

import (
	"net"
	"fmt"
	"os"
	"os/exec"
)

func Up() {
	var n, bn net.IPNet
	var i net.IP
	var s string

	/* Supplied by OpenVPN Provider */

	ozdebug := os.Getenv("oz_debug")

	ipgwstr := os.Getenv("route_network_1")
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

	i = net.ParseIP(ipgwstr)
	mask := net.CIDRMask(24, 32)
	i = i.Mask(mask)
	n.Mask = mask
	n.IP = i

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
		fmt.Fprintf(os.Stderr,s)
	}
	cmd.Run()
}
