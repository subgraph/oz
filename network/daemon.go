package network

import(
	//Builtin
	"errors"
	"fmt"
	//"math"
	"net"
	"os"
	//"os/exec"
	"strings"

	//Internal

	//External
	"github.com/milosgajdos83/tenus"
	"github.com/op/go-logging"
)


func BridgeInit(log *logging.Logger) (*HostNetwork, error) {
	htn := new(HostNetwork)
	
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use netinit from child."))
	}

	// Fetch the bridge interface by ifname
	brL, err := net.InterfaceByName(ozDefaultInterfaceBridge)
	if err != nil {
		log.Info("Bridge not found, attempting to create a new one")

		_, err = createNewBridge(log)
		if err != nil {
			return nil, fmt.Errorf("Unable to create bridge %+v", err)
		}

		// Load the new interface
		brL, _ = net.InterfaceByName(ozDefaultInterfaceBridge)
	} else {
		log.Info("Bridge already exists attempting to reuse it")
	}

	// Lookup the bridge ip addresses
	addrs, _ := brL.Addrs()
	if len(addrs) == 0 {
		return nil, errors.New("Host bridge does not have an IP address assigned")
	}

	// Try to build the network config from the bridge's address
	addrIndex := -1
	for i, addr := range addrs {
		bIP, brIP, _ := net.ParseCIDR(addr.String())

		// Discard IPv6 (TODO...)
		if bIP.To4() != nil {
			addrIndex = i

			bMask := []byte(brIP.Mask)

			htn.netmask = net.IPv4(bMask[0], bMask[1], bMask[2], bMask[3])
			htn.gateway = net.ParseIP(strings.Split(addr.String(), "/")[0])
			htn.class = strings.Split(addr.String(), "/")[1]
			htn.broadcast = net_getbroadcast(bIP, brIP.Mask)

			htn.min = inet_aton(bIP)
			htn.min++

			htn.max = inet_aton(htn.broadcast)
			htn.max--

			break
		}

	}

	if addrIndex < 0 {
		return nil, errors.New("Could not find IPv4 for bridge interface")
	}

	return htn, nil
}


// Create a new bridge on the host
// Assign it an unused range
// And bring the interface up
func createNewBridge(log *logging.Logger) (tenus.Bridger, error) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use netinit from child."))
	}

	// Create the bridge
	br, err := tenus.NewBridgeWithName(ozDefaultInterfaceBridge)
	if err != nil {
		return nil, err
	}

	// Lookup an empty ip range
	brIp, brIpNet, err := findEmptyRange(log)
	if err != nil {
		return nil, errors.New("Could not find an ip range to assign to the bridge")
	}

	// Setup the bridge's address
	if err := br.SetLinkIp(brIp, brIpNet); err != nil {
		return nil, err
	}

	// Bridge the interface up
	if err = br.SetLinkUp(); err != nil {
		return nil, fmt.Errorf("Unable to bring bridge '%+v' up: %+v", ozDefaultInterfaceBridge, err)
	}

	return br, nil

}


// Look at all the assigned IP address and try to find an available range
// Returns a ip range in the CIDR form if found or an empty string
func findEmptyRange(log *logging.Logger) (net.IP, *net.IPNet, error) {
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

			availableRange = inet_ntoa(bMin).String() + "/24"

			log.Info("Found available range: %+v", availableRange)

			return net.ParseCIDR(availableRange)
		}

	}

	return nil, nil, errors.New("Could not find an available range")

}
