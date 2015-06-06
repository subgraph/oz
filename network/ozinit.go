package network

import (
	//Builtin
	"errors"
	"fmt"
	//"math"
	"math/rand"
	"net"
	"os"
	"time"

	//External
	"github.com/j-keck/arping"
	"github.com/milosgajdos83/tenus"
	"github.com/op/go-logging"
)

func NetInit(log *logging.Logger, htn *HostNetwork, childPid int) (*SandboxNetwork, error) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use netSetup from child."))
	}
	
	stn := new(SandboxNetwork)
	
	stn.vethHost = tenus.MakeNetInterfaceName(ozDefaultInterfacePrefix)
	stn.vethGuest = stn.vethHost + "1"

	// Seed random number generator (poorly but we're not doing crypto)
	rand.Seed(time.Now().Unix() ^ int64((os.Getpid() + childPid)))

	log.Info("Configuring host veth pair: %s", stn.vethHost)

	// Fetch the bridge from the ifname
	br, err := tenus.BridgeFromName(ozDefaultInterfaceBridge)
	if err != nil {
		return nil, fmt.Errorf("Unable to attach to bridge interface %, %s.", ozDefaultInterfaceBridge, err)
	}

	// Create the veth pair
	veth, err := tenus.NewVethPairWithOptions(stn.vethHost, tenus.VethOptions{PeerName: stn.vethGuest})
	if err != nil {
		return nil, fmt.Errorf("Unable to create veth pair %s, %s.", stn.vethHost, err)
	}

	// Fetch the newly created hostside veth
	vethIf, err := net.InterfaceByName(stn.vethHost)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch veth pair %s, %s.", stn.vethHost, err)
	}

	// Add the host side veth to the bridge
	if err = br.AddSlaveIfc(vethIf); err != nil {
		return nil, fmt.Errorf("Unable to add veth pair %s to bridge, %s.", stn.vethHost, err)
	}

	// Bring the host side veth interface up
	if err = veth.SetLinkUp(); err != nil {
		return nil, fmt.Errorf("Unable to bring veth pair %s up, %s.", stn.vethHost, err)
	}

	// Assign the veth path to the namespace
	pid := childPid
	if err := veth.SetPeerLinkNsPid(pid); err != nil {
		return nil, fmt.Errorf("Unable to add veth pair %s to namespace, %s.", stn.vethHost, err)
	}

	// Allocate a new IP address
	stn.ip = getFreshIP(htn.min, htn.max, log)
	if stn.ip == "" {
		return nil, errors.New("Unable to acquire random IP")
	}

	// Parse the ip/class into the the appropriate formats
	vethGuestIp, vethGuestIpNet, err := net.ParseCIDR(stn.ip + "/" + htn.class)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse ip %s, %s.", stn.ip, err)
	}

	// Set interface address in the namespace
	if err := veth.SetPeerLinkNetInNs(pid, vethGuestIp, vethGuestIpNet, nil); err != nil {
		return nil, fmt.Errorf("Unable to parse ip link in namespace, %s.", err)
	}

	return stn, nil

}

// Setup the networking inside the child
// Namely setup the loopback interface
// and the veth interface if requested
func NetSetup(stn *SandboxNetwork, htn *HostNetwork) error {
	if os.Getpid() != 1 {
		panic(errors.New("Cannot use netChildSetup from child."))
	}

	// Bring loopback interface up
	lo, err := tenus.NewLinkFrom("lo")
	if err != nil {
		return fmt.Errorf("Unable to fetch loopback interface, %s.", err)
	}

	// Bring the link up
	err = lo.SetLinkUp()
	if err != nil {
		return fmt.Errorf("Unable to bring loopback interface up, %s.", err)
	}

	// If required configure veth
	if stn.vethGuest != "" {
		ifc, err := tenus.NewLinkFrom(stn.vethGuest)
		if err == nil {
			// Bring the link down to prepare for renaming
			if err = ifc.SetLinkDown(); err != nil {
				return fmt.Errorf("Unable to bring interface %s down, %s.", stn.vethGuest, err)
			}

			// Rename the interface to a standard eth0 (not really a necessity)
			if err = tenus.RenameInterface(stn.vethGuest, ozDefaultInterfaceInternal); err != nil {
				return fmt.Errorf("Unable to rename interface %s, %s.", stn.vethGuest, err)
			}

			// Refetch the interface again as it has changed
			ifc, err = tenus.NewLinkFrom(ozDefaultInterfaceInternal)
			if err != nil {
				return fmt.Errorf("Unable to fetch interface %s, %s.", ozDefaultInterfaceInternal, err)
			}

			// Bring the link back up
			if err = ifc.SetLinkUp(); err != nil {
				return fmt.Errorf("Unable to bring interface %s up, %s.", ozDefaultInterfaceInternal, err)
			}

			// Set the link's default gateway
			if err = ifc.SetLinkDefaultGw(&htn.gateway); err != nil {
				return fmt.Errorf("Unable to set default route %s.", err)
			}

		} else {
			return fmt.Errorf("Unable to fetch inteface %s, %s.", stn.vethGuest, err)
		}
	}

	return nil

}

// Try to find an unassigned IP address
// Do this by first trying two random IPs or, if that fails, sequentially
func getFreshIP(min, max uint64, log *logging.Logger) string {
	var newIP string

	for i := 0; i < ozMaxRandTries; i++ {
		newIP = getRandIP(min, max)
		if newIP != "" {
			break
		}
	}

	if newIP == "" {
		log.Notice("Random IP lookup failed %d times, reverting to sequential select", ozMaxRandTries)

		newIP = getScanIP(min, max)
	}

	return newIP

}

// Generate a random ip and arping it to see if it is available
// Returns the ip on success or an ip string is the ip is already taken
func getRandIP(min, max uint64) string {
	if min > max {
		return ""
	}

	dstIP := inet_ntoa(uint64(rand.Int63n(int64(max-min))) + min)

	arping.SetTimeout(time.Millisecond * 150)

	_, _, err := arping.PingOverIfaceByName(dstIP, ozDefaultInterfaceBridge)

	if err == arping.ErrTimeout {
		return dstIP.String()
	} else if err != nil {
		return dstIP.String()
	}

	return ""

}

// Go through all possible ips between min and max
// and arping each one until a free one is found
func getScanIP(min, max uint64) string {
	if min > max {
		return ""
	}

	for i := min; i < max; i++ {
		dstIP := inet_ntoa(i)

		arping.SetTimeout(time.Millisecond * 150)

		_, _, err := arping.PingOverIfaceByName(dstIP, ozDefaultInterfaceBridge)

		if err == arping.ErrTimeout {
			return dstIP.String()
		} else if err != nil {
			return dstIP.String()
		}
	}

	return ""

}
