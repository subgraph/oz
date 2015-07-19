package network

import (
	//Builtin
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/subgraph/oz/ns"

	"github.com/op/go-logging"
)

type ProxyType string

const (
	PROXY_CLIENT ProxyType = "client"
	PROXY_SERVER ProxyType = "server"
)

type ProtoType string

const (
	PROTO_TCP    ProtoType = "tcp"
	PROTO_UDP    ProtoType = "udp"
	PROTO_SOCKET ProtoType = "socket"
)

// Socket list, used to hold ports that should be forwarded
type ProxyConfig struct {
	// One of client, server
	Nettype ProxyType `json:"type"`

	// One of tcp, udp, socket
	Proto ProtoType

	// TCP or UDP port number
	Port int

	// Unix socket to attach to
	//  applies to proto: socket only
	Socket string

	// Optional: Destination address
	// In client mode: the host side address to connect to
	// In server mode: the sandbox side address to bind to
	// If left empty, localhost is used
	Destination string
}

var wgProxy sync.WaitGroup

func ProxySetup(childPid int, ozSockets []ProxyConfig, log *logging.Logger, ready sync.WaitGroup) error {
	for _, socket := range ozSockets {
		if socket.Nettype == "" || socket.Nettype == PROXY_CLIENT {
			err := newProxyClient(childPid, socket.Proto, socket.Destination, socket.Port, log, ready)
			if err != nil {
				return fmt.Errorf("Unable to setup client socket forwarding %+v, %s", socket, err)
			}
		} else if socket.Nettype == PROXY_SERVER {
			err := newProxyServer(childPid, socket.Proto, socket.Destination, socket.Port, log, ready)
			if err != nil {
				return fmt.Errorf("Unable to setup server socket forwarding %+s, %s", socket, err)
			}
		}
	}

	return nil
}

/**
 * Listener/Client
**/
func proxyClientConn(conn *net.Conn, proto ProtoType, rAddr string, ready sync.WaitGroup) error {
	rConn, err := net.Dial(string(proto), rAddr)
	if err != nil {
		return fmt.Errorf("Socket: %+v.\n", err)
	}

	go io.Copy(rConn, *conn)
	go io.Copy(*conn, rConn)

	return nil
}

func newProxyClient(pid int, proto ProtoType, dest string, port int, log *logging.Logger, ready sync.WaitGroup) error {
	if dest == "" {
		dest = "127.0.0.1"
	}

	lAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	rAddr := net.JoinHostPort(dest, strconv.Itoa(port))

	log.Info("Starting socket client forwarding: %s://%s.", proto, rAddr)

	listen, err := proxySocketListener(pid, proto, lAddr)
	if err != nil {
		return err
	}

	wgProxy.Add(1)
	go func() {
		defer wgProxy.Done()
		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Error("Socket: %+v.", err)
				//panic(err)
				continue
			}

			go proxyClientConn(&conn, proto, rAddr, ready)
		}
	}()

	return nil
}

func proxySocketListener(pid int, proto ProtoType, lAddr string) (net.Listener, error) {
	fd, err := ns.OpenProcess(pid, ns.CLONE_NEWNET)
	defer ns.Close(fd)
	if err != nil {
		return nil, err
	}

	return nsSocketListener(fd, proto, lAddr)
}

func nsSocketListener(fd uintptr, proto ProtoType, lAddr string) (net.Listener, error) {
	origNs, _ := ns.OpenProcess(os.Getpid(), ns.CLONE_NEWNET)
	defer ns.Close(origNs)
	defer ns.Set(origNs, ns.CLONE_NEWNET)

	err := ns.Set(uintptr(fd), ns.CLONE_NEWNET)
	if err != nil {
		return nil, err
	}

	return net.Listen(string(proto), lAddr)

}

/**
 * Connect/Server
**/
func proxyServerConn(pid int, conn *net.Conn, proto ProtoType, rAddr string, log *logging.Logger, ready sync.WaitGroup) error {
	rConn, err := socketConnect(pid, proto, rAddr)
	if err != nil {
		log.Error("Socket: %+v.", err)
		return err
	}

	go io.Copy(*conn, rConn)
	go io.Copy(rConn, *conn)

	return nil
}

func newProxyServer(pid int, proto ProtoType, dest string, port int, log *logging.Logger, ready sync.WaitGroup) error {
	if dest == "" {
		dest = "127.0.0.1"
	}

	lAddr := net.JoinHostPort(dest, strconv.Itoa(port))
	rAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	log.Info("Starting socket server forwarding: %s://%s.", proto, lAddr)

	listen, err := net.Listen(string(proto), lAddr)
	if err != nil {
		return err
	}

	wgProxy.Add(1)
	go func() {
		defer wgProxy.Done()
		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Error("Socket: %+v.", err)
				//panic(err)
				continue
			}

			go proxyServerConn(pid, &conn, proto, rAddr, log, ready)
		}
	}()

	return nil
}

func socketConnect(pid int, proto ProtoType, rAddr string) (net.Conn, error) {
	fd, err := ns.OpenProcess(pid, ns.CLONE_NEWNET)
	defer ns.Close(fd)
	if err != nil {
		return nil, err
	}

	return nsProxySocketConnect(fd, proto, rAddr)
}

func nsProxySocketConnect(fd uintptr, proto ProtoType, rAddr string) (net.Conn, error) {
	origNs, _ := ns.OpenProcess(os.Getpid(), ns.CLONE_NEWNET)
	defer ns.Close(origNs)
	defer ns.Set(origNs, ns.CLONE_NEWNET)

	err := ns.Set(uintptr(fd), ns.CLONE_NEWNET)
	if err != nil {
		return nil, err
	}

	return net.Dial(string(proto), rAddr)

}
