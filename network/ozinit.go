package network

import (
	//Builtin
	"errors"
	"fmt"
	"os"

	// Internal
	//"github.com/op/go-logging"

	//External
	"github.com/milosgajdos83/tenus"
)

// Setup the networking inside the child
// Namely setup the loopback interface
// and the veth interface if requested
func NetSetup(stn *SandboxNetwork) error {
	if os.Getpid() != 1 {
		panic(errors.New("Cannot use NetSetup from parent."))
	}

	if err := setupLoopback(stn); err != nil {
		return fmt.Errorf("Unable to setup loopback interface: %+v", err)
	}

	// If required configure veth
	if stn.VethGuest != "" {
		if err := setupVEth(stn); err != nil {
			return fmt.Errorf("Unable to setup veth interface: %+v", err)
		}
	}

	return nil
}

func (stn *SandboxNetwork) NetReconfigure() error {
	if os.Getpid() != 1 {
		panic(errors.New("Cannot use NetReconfigure from parent."))
	}

	// Set the link's default gateway
	if err := stn.Interface.SetLinkDefaultGw(&stn.Gateway); err != nil {
		return fmt.Errorf("Unable to set default route %s.", err)
	}

	return nil
}

func setupLoopback(stn *SandboxNetwork) error {
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

	return nil
}

func setupVEth(stn *SandboxNetwork) error {
	ifc, err := tenus.NewLinkFrom(stn.VethGuest)

	if err != nil {
		return fmt.Errorf("Unable to fetch inteface %s, %s.", stn.VethGuest, err)
	}

	// Bring the link down to prepare for renaming
	if err = ifc.SetLinkDown(); err != nil {
		return fmt.Errorf("Unable to bring interface %s down, %s.", stn.VethGuest, err)
	}

	// Rename the interface to a standard eth0 (not really a necessity)
	if err = tenus.RenameInterface(stn.VethGuest, ozDefaultInterfaceInternal); err != nil {
		return fmt.Errorf("Unable to rename interface %s, %s.", stn.VethGuest, err)
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
	if err = ifc.SetLinkDefaultGw(&stn.Gateway); err != nil {
		return fmt.Errorf("Unable to set default route %s.", err)
	}

	stn.Interface = ifc

	return nil
}
