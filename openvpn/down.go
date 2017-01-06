package openvpn

import (
	"net"
	"os"
	"os/exec"
)

func Down() {
	var n net.IPNet
	var bn net.IPNet

	ipgwstr := os.Getenv("route_network_1")
	iflstr := os.Getenv("ifconfig_local")

	bridgeaddr := os.Getenv("bridge_addr")
	table := os.Getenv("routing_table")

	i := net.ParseIP(ipgwstr)
	mask := net.CIDRMask(24, 32)
	i = i.Mask(mask)
	n.Mask = mask
	n.IP = i

	bi := net.ParseIP(bridgeaddr)
	bmask := net.CIDRMask(24, 32)
	bi = bi.Mask(bmask)
	bn.Mask = bmask
	bn.IP = bi

	/* Drop routing rules */

	//	s := fmt.Sprintf("ip route flush table %s\n", table)
	cmd := exec.Command("/bin/ip", "route", "flush", "table", table)
	cmd.Run()

	/* Drop policy rules */

	//	s += fmt.Sprintf("ip rule del from all to %s lookup %s\n", iflstr, table)
	cmd = exec.Command("/bin/ip", "rule", "del", "from", "all", "to", iflstr, "lookup", table)
	cmd.Run()

	//	s += fmt.Sprintf("ip rule del from %s lookup %s\n", iflstr, table)
	cmd = exec.Command("/bin/ip", "rule", "del", "from", iflstr, "lookup", table)
	cmd.Run()

	//	s += fmt.Sprintf("ip rule del from %s lookup %s\n", bn.String(), table)
	cmd = exec.Command("/bin/ip", "rule", "del", "from", bn.String(), "lookup", table)
	cmd.Run()

	//	s += fmt.Sprintf("ip rule add from all to %s lookup %s\n", bn.String(), table)
	cmd = exec.Command("/bin/ip", "rule", "del", "from", "all", "to", bn.String(), "lookup", table)
	cmd.Run()

}
