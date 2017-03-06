package network

import (
	"errors"
	"fmt"
	"github.com/milosgajdos83/tenus"
	"github.com/op/go-logging"
	"net"
)

// Bridges manages the creation of bridges for sandbox bridged networking
type Bridges struct {
	log         *logging.Logger      // global logger
	initialized bool                 // Initialize the following fields lazily
	alloc       *subnetAllocator     // allocates subnet ranges for new bridges
	bridgeMap   map[string]*OzBridge // Map of names to bridge instances
}

// OzBridge represents a single bridge used for sandbox bridged networking
type OzBridge struct {
	tenus.Bridger                 // Bridge instance
	Name          string          // Name of bridge
	ipr           *IPRange        // IPRange for allocating addresses to veth interfaces
	ip            *net.IP         // IP assigned to the bridge itself
	veths         map[int]*OzVeth // map from sandbox id to OzVeth instances
	log           *logging.Logger
}

// OzVeth is a pair of Veth interfaces
type OzVeth struct {
	tenus.Vether           // The pair of veth interfaces
	id           int       // The id of the sandbox this veth pair is associated with
	peerPid      int       // The process id of the init process of the sandbox this veth pair belongs to
	bridge       *OzBridge // The bridge this veth pair is attached to
	log          *logging.Logger
}

func (b *OzBridge) configure() error {
	ip := b.ipr.FirstIP()
	b.ip = &ip 
	b.log.Infof("Configuring bridge %s with IP address %v", b.Name, ip)
	if err := b.SetLinkDown(); err != nil {
		return fmt.Errorf("error bringing interface down: %v", err)
	}
	if err := b.SetLinkIp(ip, b.ipr.IPNet); err != nil {
		return fmt.Errorf("error configuring IP address of bridge: %v", err)
	}
	if err := b.SetLinkUp(); err != nil {
		return fmt.Errorf("error bringing bridge interface up: %v", err)
		return err
	}
	return nil
}

func (b *OzBridge) reconfigure(ipr *IPRange) error {
	if err := b.SetLinkIp(ipr.FirstIP(), ipr.IPNet); err != nil {

	}
	b.ipr = ipr
	for _, veth := range b.veths {
		if err := veth.AssignIP(); err != nil {
			return err
		}
	}
	return nil
}

func (b *OzBridge) NewVeth(id int, peerPid int) (*OzVeth, error) {
	if b.veths[id] != nil {
		return nil, fmt.Errorf("a veth already exists on this bridge for id=%d", id)
	}
	v, err := b.newVeth(id, peerPid)
	if err != nil {
		return nil, err
	}
	b.veths[id] = v
	return v, nil
}

func (b *OzBridge) newVeth(id int, peerPid int) (*OzVeth, error) {
	vpair, err := createVethPair()
	if err != nil {
		return nil, err
	}
	b.log.Infof("Created veth pair %s/%s", vpair.NetInterface().Name, vpair.PeerNetInterface().Name)
	return &OzVeth{
		Vether:  vpair,
		id:      id,
		peerPid: peerPid,
		bridge:  b,
		log:     b.log,
	}, nil
}

func (b *OzBridge) GetIP() (*net.IP) {
	return b.ip
}

func createVethPair() (tenus.Vether, error) {
	hostName := tenus.MakeNetInterfaceName(ozDefaultInterfacePrefix)
	guestName := hostName + "1"
	veth, err := tenus.NewVethPairWithOptions(hostName, tenus.VethOptions{PeerName: guestName})
	if err != nil {
		return nil, fmt.Errorf("failed to create veth pair %s/%s: %v", hostName, guestName, err)
	}
	return veth, nil
}

func (v *OzVeth) Setup() error {
	if err := v.bridge.AddSlaveIfc(v.NetInterface()); err != nil {
		return fmt.Errorf("failed to add veth %s to bridge: %v", v.NetInterface().Name, err)
	}

	if err := v.SetLinkUp(); err != nil {
		return fmt.Errorf("failed to bring host veth %s up: %v", v.NetInterface().Name, err)
	}

	if err := v.SetPeerLinkNsPid(v.peerPid); err != nil {
		return fmt.Errorf("failed to send peer veth %s into sandbox (pid: %d): %v", v.PeerNetInterface().Name, v.peerPid, err)
	}

	if err := v.AssignIP(); err != nil {
		return fmt.Errorf("failed to assign address to peer veth %s: %v", v.PeerNetInterface().Name, err)
	}
	return nil
}

func (v *OzVeth) AssignIP() error {
	ip := v.bridge.ipr.FreshIP()

	v.log.Infof("Assigning IP address %v to sandbox veth %s", ip, v.PeerNetInterface().Name)

	if ip == nil {
		return errors.New("unable to find usable IP address")
	}
	return v.SetIP(ip)
}

func (v *OzVeth) SetIP(ip net.IP) error {
	ipnet := v.bridge.ipr
	gw := v.bridge.ip
	return v.SetPeerLinkNetInNs(v.peerPid, ip, ipnet.IPNet, gw)
}

func (v *OzVeth) Delete() error {
	return v.DeleteLink()
}

func NewBridges(log *logging.Logger) *Bridges {
	return &Bridges{
		log: log,
	}
}

func (bs *Bridges) ensureInitialized() error {
	if bs.initialized {
		return nil
	}
	a, err := newAllocator(bs.log)
	if err != nil {
		return err
	}
	bs.alloc = a
	bs.bridgeMap = make(map[string]*OzBridge)
	bs.initialized = true
	return nil
}

func (bs *Bridges) GetBridgeMap() (map[string]*OzBridge) {
	return bs.bridgeMap
}
 
func (bs *Bridges) GetBridge(name string) (*OzBridge, error) {
	if err := bs.ensureInitialized(); err != nil {
		return nil, err
	}

	if bs.bridgeMap[name] == nil {
		br, err := bs.createBridge(name)
		if err != nil {
			return nil, err
		}
		if err := br.configure(); err != nil {
			return nil, err
		}
		bs.bridgeMap[name] = br
	}
	return bs.bridgeMap[name], nil
}

func (bs *Bridges) createBridge(name string) (*OzBridge, error) {
	brname := ozDefaultInterfaceBridgeBase + name
	bs.log.Infof("Creating new bridge '%s'", brname)
	br, err := bs.openBridge(brname)
	if err != nil {
		return nil, err
	}
	r, err := bs.alloc.allocateRange(brname)
	if err != nil {
		return nil, err
	}
	return &OzBridge{
		Bridger: br,
		Name:    name,
		ipr:     r,
		veths:   make(map[int]*OzVeth),
		log:     bs.log,
	}, nil
}

func (bs *Bridges) openBridge(brname string) (tenus.Bridger, error) {
	br, err := tenus.BridgeFromName(brname)
	if err == nil {
		bs.log.Infof("Bridge '%s' already exists, deleting it.", brname)
		if err := br.DeleteLink(); err != nil {
			bs.log.Warningf("error deleting bridge: %v", err)
		}
	}
	return tenus.NewBridgeWithName(brname)
}

func (bs *Bridges) Reconfigure() error {
	if !bs.initialized || !bs.alloc.needsReconfigure() {
		return nil
	}

	a, err := newAllocator(bs.log)
	if err != nil {
		return err
	}
	bs.alloc = a

	for _, ozb := range bs.bridgeMap {
		brname := ozDefaultInterfaceBridgeBase + ozb.Name
		ipr, err := bs.alloc.allocateRange(brname)
		if err != nil {
			return err
		}
		if err := ozb.reconfigure(ipr); err != nil {
			return err
		}
	}
	return nil
}

func (v *OzVeth) GetVethBridge() *OzBridge {
	return v.bridge
}
