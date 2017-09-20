package network

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
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
	PROTO_TCP         ProtoType = "tcp"
	PROTO_UDP         ProtoType = "udp"
	PROTO_UNIX        ProtoType = "unix"
	PROTO_TCP_TO_UNIX ProtoType = "tcp2unix"
	PROTO_UNIXGRAM    ProtoType = "unixgram"
	PROTO_UNIXPACKET  ProtoType = "unixpacket"
)

// Socket list, used to hold ports that should be forwarded
type ProxyConfig struct {
	// One of client, server
	Nettype ProxyType `json:"type"`

	// One of tcp, udp, socket
	Proto ProtoType

	// TCP or UDP port number
	Port int

	// Destination port number
	DPort int
	// Optional: Destination address
	// In client mode: the host side address to connect to
	// In server mode: the sandbox side address to bind to
	// For unix sockets this is an abstract path
	// If left empty, localhost is used
	Destination string
}

var wgProxy sync.WaitGroup

type PConnInfo struct {
	Saddr net.IP
	Sport uint16
	Daddr net.IP
	Dport uint16
}

type ProxyPair struct {
	In  *PConnInfo
	Out *PConnInfo
	Cnt int
}

var ProxyPairs []*ProxyPair
var PairLock = &sync.Mutex{}

func connToPConn(c net.Conn, swap bool) *PConnInfo {
	rstr := c.LocalAddr().String()
	lstr := c.RemoteAddr().String()

	if swap {
		tmp := rstr
		rstr = lstr
		lstr = tmp
	}

	rhosts, rports, err := net.SplitHostPort(rstr)
	lhosts, lports, err2 := net.SplitHostPort(lstr)

	if err == nil && err2 == nil {
		srci := net.ParseIP(lhosts)
		dsti := net.ParseIP(rhosts)
		sport, err := strconv.Atoi(lports)
		dport, err2 := strconv.Atoi(rports)

		if srci != nil && dsti != nil && err == nil && err2 == nil {

			if sport >= 0 && sport <= 65535 && dport >= 0 && dport <= 65535 {
				return &PConnInfo{Saddr: srci, Sport: uint16(sport), Daddr: dsti, Dport: uint16(dport)}
			}

		}
	}

	return nil
}

func GetProxyPairInfo() []string {
	PairLock.Lock()
	defer PairLock.Unlock()

	result := make([]string, 0)

	for _, pair := range ProxyPairs {
		rstr := fmt.Sprintf("%v:%d -> %v:%d, %v:%d -> %v:%d", pair.In.Saddr, pair.In.Sport, pair.In.Daddr, pair.In.Dport,
			pair.Out.Saddr, pair.Out.Sport, pair.Out.Daddr, pair.Out.Dport)
		result = append(result, rstr)
	}

	return result
}

func addProxyPair(in net.Conn, out net.Conn, swap bool) bool {
	PairLock.Lock()
	defer PairLock.Unlock()
	pin := connToPConn(in, false)
	pout := connToPConn(out, swap)

	if pin == nil || pout == nil {
		return false
	}

	ProxyPairs = append(ProxyPairs, &ProxyPair{In: pin, Out: pout, Cnt: 2})
	return true
}

func pConnEqual(pair1, pair2 *PConnInfo, loose bool) bool {
	if loose && (pair1.Saddr.Equal(pair2.Daddr) && pair1.Sport == pair2.Dport && pair1.Daddr.Equal(pair2.Saddr) && pair1.Dport == pair2.Sport) {
		return true
	}

	return pair1.Saddr.Equal(pair2.Saddr) && pair1.Sport == pair2.Sport && pair1.Daddr.Equal(pair2.Daddr) && pair1.Dport == pair2.Dport
}

func removeProxyPair(in net.Conn, out net.Conn) bool {
	PairLock.Lock()
	defer PairLock.Unlock()

	pin := connToPConn(in, false)
	pout := connToPConn(out, false)

	if pin == nil || pout == nil {
		return false
	}

	for i, pair := range ProxyPairs {
		if pConnEqual(pair.In, pin, false) && pConnEqual(pair.Out, pout, true) {
			pair.Cnt--

			if pair.Cnt <= 0 {
				ProxyPairs = append(ProxyPairs[:i], ProxyPairs[i+1:]...)
			}

			return true
		}
	}

	return false
}

func ProxySetup(childPid int, ozSockets []ProxyConfig, log *logging.Logger, ready sync.WaitGroup) error {
	for _, socket := range ozSockets {
		if socket.Nettype == "" {
			continue
		}
		if socket.Nettype == PROXY_CLIENT {
			err := newProxyClient(childPid, &socket, log, ready)
			if err != nil {
				return fmt.Errorf("%+v, %s", socket, err)
			}
		} else if socket.Nettype == PROXY_SERVER {
			err := newProxyServer(childPid, &socket, log, ready)
			if err != nil {
				return fmt.Errorf("%+s, %s", socket, err)
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

	var wg sync.WaitGroup
	wg.Add(2)

	copyLoop := func(dst, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		defer removeProxyPair(*conn, rConn)
		io.Copy(dst, src)
	}

	//	fmt.Println("XXX: attempting to add proxy client pair...")
	if !addProxyPair(*conn, rConn, true) {
		fmt.Println("Could not add new proxy client pair to table.")
	}

	go copyLoop(*conn, rConn)
	go copyLoop(rConn, *conn)

	return nil
}

func newProxyClient(pid int, config *ProxyConfig, log *logging.Logger, ready sync.WaitGroup) error {
	if config.Destination == "" {
		config.Destination = "127.0.0.1"
	}

	var lAddr, rAddr, dport string
	if strings.HasPrefix(string(config.Proto), "tcp") && config.Proto != PROTO_TCP_TO_UNIX {
		if config.DPort != 0 {
			dport = strconv.Itoa(config.DPort)
		} else {
			dport = strconv.Itoa(config.Port)
		}
		lAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(config.Port))
		rAddr = net.JoinHostPort(config.Destination, dport)
	} else if strings.HasPrefix(string(config.Proto), "unix") {
		if !strings.HasPrefix(config.Destination, "@") {
			log.Warning("Only abstract unix socket are supported!")
			return nil
		}
		lAddr = config.Destination
		rAddr = config.Destination
	} else if config.Proto == PROTO_TCP_TO_UNIX {
		lAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(config.Port))
		rAddr = config.Destination
	} else {
		log.Warning("Unsupported proxy protocol specified!")
		return nil
	}

	var listenProto ProtoType
	if config.Proto == PROTO_TCP_TO_UNIX {
		listenProto = PROTO_TCP
		log.Info("Starting socket client forwarding: %s://%s -> unix://%s.", listenProto, lAddr, config.Destination)
	} else {
		listenProto = config.Proto
		log.Info("Starting socket client forwarding: %s://%s.", listenProto, rAddr)
	}
	listen, err := proxySocketListener(pid, listenProto, lAddr)
	if err != nil {
		return err
	}

	wgProxy.Add(1)
	c := *config

	go func() {
		defer wgProxy.Done()
		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Error("Socket: %+v.", err)
				//panic(err)
				continue
			}
			/*
				if err = conn.SetDeadline(time.Now().Add(50*time.Second)); err != nil {
					log.Error("conn: %+v", err)
					continue
				}
				defer func() {
					// Disarm the handshake timeout, only propagate the error if
					// the handshake was successful.
					nerr := conn.SetDeadline(time.Time{})
					if err == nil {
						err = nerr
					}
				}()
			*/
			/*	req := new(Request)
				req.conn = conn
			*/

			var dialProto ProtoType
			if c.Proto == PROTO_TCP_TO_UNIX {
				dialProto = PROTO_UNIX
			} else {
				dialProto = c.Proto
			}

			go proxyClientConn(&conn, dialProto, rAddr, ready)
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

	var wg sync.WaitGroup
	wg.Add(2)

	copyLoop := func(dst, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		io.Copy(dst, src)
	}

	//	log.Error("XXX: attempting to add proxy server pair...")
	/*	if !addProxyPair(*conn, rConn, false) {
		log.Error("Could not add new proxy server pair to table.")
	} */

	go copyLoop(*conn, rConn)
	go copyLoop(rConn, *conn)

	return nil
}

func newProxyServer(pid int, config *ProxyConfig, log *logging.Logger, ready sync.WaitGroup) error {
	if config.Destination == "" {
		config.Destination = "127.0.0.1"
	}

	var lAddr, rAddr string
	if !strings.HasPrefix(string(config.Proto), "unix") {
		lAddr = net.JoinHostPort(config.Destination, strconv.Itoa(config.Port))
		rAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(config.Port))
	} else {
		if !strings.HasPrefix(config.Destination, "@") {
			log.Warning("Only abstract unix socket are supported!")
			return nil
		}
		lAddr = config.Destination
		rAddr = config.Destination
	}

	log.Info("Starting socket server forwarding: %s://%s.", config.Proto, lAddr)

	listen, err := net.Listen(string(config.Proto), lAddr)
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

			go proxyServerConn(pid, &conn, config.Proto, rAddr, log, ready)
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
