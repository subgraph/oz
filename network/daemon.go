package network

import(
	//Builtin
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	//Internal
	"github.com/op/go-logging"

	//External
	"github.com/milosgajdos83/tenus"
)

func BridgeInit(bridgeMAC string, nmIgnoreFile string, log *logging.Logger) (*HostNetwork, error) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use netinit from child."))
	}
	
	htn := &HostNetwork{
		BridgeMAC: bridgeMAC,
	}
	
	if _, err := os.Stat(nmIgnoreFile); os.IsNotExist(err) {
		log.Warning("Warning! Network Manager may not properly configured to ignore the bridge interface! This may result in management conflicts!")
	}
	
	br, err := tenus.BridgeFromName(ozDefaultInterfaceBridge)
	if err != nil {
		log.Info("Bridge not found, attempting to create a new one")
		
		br, err = tenus.NewBridgeWithName(ozDefaultInterfaceBridge)
		if err != nil {
			return nil, fmt.Errorf("Unable to create bridge %+v", err)
		}
	}
	
	if err:= htn.configureBridgeInterface(br, log); err != nil {
		return nil, err
	}
	
	brL := br.NetInterface()
	addrs, err := brL.Addrs()
	if err != nil {
		return nil, fmt.Errorf("Unable to get bridge interface addresses: %+v", err)
	}
	
	// Build the ip range which we will use for the network
	if err := htn.buildBridgeNetwork(addrs); err != nil {
		return nil, err
	}

	return htn, nil

}

func PrepareSandboxNetwork(htn *HostNetwork, log *logging.Logger) (*SandboxNetwork, error) {
	stn := new(SandboxNetwork)
	
	stn.VethHost = tenus.MakeNetInterfaceName(ozDefaultInterfacePrefix)
	stn.VethGuest = stn.VethHost + "1"
	
	stn.Gateway = htn.Gateway
	stn.Class = htn.Class
	
	// Allocate a new IP address
	stn.Ip = getFreshIP(htn.Min, htn.Max, log)
	if stn.Ip == "" {
		return nil, errors.New("Unable to acquire random IP")
	}
	
	return stn, nil
}

func NetInit(stn *SandboxNetwork, htn *HostNetwork, childPid int, log *logging.Logger) error {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use netSetup from child."))
	}
	
	// Seed random number generator (poorly but we're not doing crypto)
	rand.Seed(time.Now().Unix() ^ int64((os.Getpid() + childPid)))

	log.Info("Configuring host veth pair '%s' with: %s", stn.VethHost, stn.Ip + "/" + htn.Class)

	// Fetch the bridge from the ifname
	br, err := tenus.BridgeFromName(ozDefaultInterfaceBridge)
	if err != nil {
		return fmt.Errorf("Unable to attach to bridge interface %, %s.", ozDefaultInterfaceBridge, err)
	}

	// Make sure the bridge is configured and the link is up
	//  This really shouldn't be needed, but Network-Manager is a PITA
	//  and even if you actualy ignore the interface there's a race
	//  between the interface being created and setting it's hwaddr
	//if err := htn.configureBridgeInterface(br, log); err != nil {
	//	return fmt.Errorf("Unable to reconfigure bridge: %+v", err)
	//}
	
	// Create the veth pair
	veth, err := tenus.NewVethPairWithOptions(stn.VethHost, tenus.VethOptions{PeerName: stn.VethGuest})
	if err != nil {
		return fmt.Errorf("Unable to create veth pair %s, %s.", stn.VethHost, err)
	}

	// Fetch the newly created hostside veth
	vethIf, err := net.InterfaceByName(stn.VethHost)
	if err != nil {
		return fmt.Errorf("Unable to fetch veth pair %s, %s.", stn.VethHost, err)
	}

	// Add the host side veth to the bridge
	if err := br.AddSlaveIfc(vethIf); err != nil {
		return fmt.Errorf("Unable to add veth pair %s to bridge, %s.", stn.VethHost, err)
	}

	// Bring the host side veth interface up
	if err := veth.SetLinkUp(); err != nil {
		return fmt.Errorf("Unable to bring veth pair %s up, %s.", stn.VethHost, err)
	}

	// Assign the veth path to the namespace
	pid := childPid
	if err := veth.SetPeerLinkNsPid(pid); err != nil {
		return fmt.Errorf("Unable to add veth pair %s to namespace, %s.", stn.VethHost, err)
	}

	// Parse the ip/class into the the appropriate formats
	vethGuestIp, vethGuestIpNet, err := net.ParseCIDR(stn.Ip + "/" + htn.Class)
	if err != nil {
		return fmt.Errorf("Unable to parse ip %s, %s.", stn.Ip, err)
	}
	
	// Set interface address in the namespace
	if err := veth.SetPeerLinkNetInNs(pid, vethGuestIp, vethGuestIpNet, nil); err != nil {
		return fmt.Errorf("Unable to parse ip link in namespace, %s.", err)
	}

	return nil

}

func (stn *SandboxNetwork) Cleanup(log *logging.Logger) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use Cleanup from child."))
	}
	
	if _, err := net.InterfaceByName(stn.VethHost); err != nil {
		log.Info("No veth found to cleanup")
		return
	}
	
	tenus.DeleteLink(stn.VethHost)
}

func (htn *HostNetwork) configureBridgeInterface(br tenus.Bridger, log *logging.Logger) error  {
	// Set the bridge mac address so it can be fucking ignored by Network-Manager.
	if htn.BridgeMAC != "" {
		if err := br.SetLinkMacAddress(htn.BridgeMAC); err != nil {
			return fmt.Errorf("Unable to set MAC address for gateway", err)
		}
	}
	
	if htn.Gateway == nil {
		// Lookup an empty ip range
		brIp, brIpNet, err := findEmptyRange()
		if err != nil {
			return fmt.Errorf("Could not find an ip range to assign to the bridge")
		}
		htn.Gateway = brIp
		htn.GatewayNet = brIpNet
		log.Info("Found available range: %+v", htn.GatewayNet.String())
	}
	
	if err := br.SetLinkIp(htn.Gateway, htn.GatewayNet); err != nil {
		if os.IsExist(err) {
			log.Info("Bridge IP appears to be already assigned")
		} else {
			return fmt.Errorf("Unable to set gateway IP", err)
		}
	}

	// Bridge the interface up
	if err := br.SetLinkUp(); err != nil {
		return fmt.Errorf("Unable to bring bridge '%+v' up: %+v", ozDefaultInterfaceBridge, err)
	}

	return nil
}

func (htn *HostNetwork)buildBridgeNetwork(addrs []net.Addr) error {
	// Try to build the network config from the bridge's address
	addrIndex := -1
	for i, addr := range addrs {
		bIP, brIP, _ := net.ParseCIDR(addr.String())

		// Discard IPv6 (TODO...)
		if bIP.To4() != nil {
			addrIndex = i

			bMask := []byte(brIP.Mask)

			htn.Netmask = net.IPv4(bMask[0], bMask[1], bMask[2], bMask[3])
			htn.Gateway = net.ParseIP(strings.Split(addr.String(), "/")[0])
			htn.Class = strings.Split(addr.String(), "/")[1]
			htn.Broadcast = net_getbroadcast(bIP, brIP.Mask)

			htn.Min = inet_aton(bIP)
			htn.Min++

			htn.Max = inet_aton(htn.Broadcast)
			htn.Max--

			break
		}

	}
	
	if addrIndex < 0 {
		return errors.New("Could not find IPv4 for bridge interface")
	}
	
	return nil
}

// Look at all the assigned IP address and try to find an available range
// Returns a ip range in the CIDR form if found or an empty string
func findEmptyRange() (net.IP, *net.IPNet, error) {
	type localNet struct {
		min uint64
		max uint64
	}

	var (
		localNets      []localNet
		availableRange string
	)

	// List all the available interfaces and their addresses
	// and calulate their network's min and max values
	ifs, _ := net.Interfaces()
	for _, netif := range ifs {
		// Disable loopback and our bridge
		if netif.Name == ozDefaultInterfaceBridge ||
			strings.HasPrefix(netif.Name, "lo") {
			continue
		}

		// Go through each address on the interface
		addrs, _ := netif.Addrs()
		for _, addr := range addrs {
			bIP, brIP, _ := net.ParseCIDR(addr.String())

			// Discard non IPv4 addresses
			if bIP.To4() != nil {
				min := inet_aton(brIP.IP)
				min++

				max := inet_aton(net_getbroadcast(bIP, brIP.Mask))
				max--

				localNets = append(localNets, localNet{min: min, max: max})
			}
		}
	}

	// Go through the list of private network ranges and
	// look for one in which we cannot find a local network
	for _, ipRange := range privateNetworkRanges {
		bIP, brIP, err := net.ParseCIDR(ipRange)
		if err != nil {
			continue
		}

		bMin := inet_aton(bIP)
		bMax := inet_aton(net_getbroadcast(bIP, brIP.Mask))

		alreadyUsed := false
		for _, add := range localNets {
			if add.min >= bMin && add.min < bMax &&
				add.max > bMin && add.max <= bMax {
				alreadyUsed = true
				break

			}
		}

		// If the range is available, grab a small slice
		if alreadyUsed == false {
			bRange := bMax - bMin
			if bRange > 0xFF {
				bMin = bMax - 0xFE
			}

			// XXX
			availableRange = inet_ntoa(bMin).String() + "/24"

			return net.ParseCIDR(availableRange)
		}

	}

	return nil, nil, errors.New("Could not find an available range")

}
