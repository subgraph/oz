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
func NetSetup() error {
	if os.Getpid() != 1 {
		panic(errors.New("Cannot use NetSetup from parent."))
	}

	if err := setupLoopback(); err != nil {
		return fmt.Errorf("Unable to setup loopback interface: %+v", err)
	}

	return nil
}

func setupLoopback() error {
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
