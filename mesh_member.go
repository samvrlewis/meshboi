package meshboi

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/samvrlewis/meshboi/tun"
	"github.com/samvrlewis/udp"
	"inet.af/netaddr"
)

const bufSize = 8192

type Peer struct {
	tunIP         netaddr.IPPort
	remoteIP      netaddr.IPPort
	lastContacted time.Time
	conn          *dtls.Conn
}

type MeshMember struct {
	rollodexIp  string
	tun         *tun.Tun
	quit        chan bool
	rolloConn   net.Conn
	myOutsideIP netaddr.IPPort

	// This manages and multiplexes a set of udp "connections" that are all on a
	// connected to a single local port.
	listener *udp.Listener

	// Map of real address to tun address
	peers map[netaddr.IPPort]*Peer

	// maybe I could just pass a udp connection here for the server connection
	// but then I also need to make one for each of the clients I want to talk to.. hmm.
	// but the session once established does last "forever". how to handle one side dropping out?
	// also i think one side of the dtls has to be server, one client. how to choose? what's the flow like?
	// maybe the client with the lowest IP address can be the server?

	// so every client will listen and accept connections then accept data through that connection
	// but if it wants to send data out it will send it as a "client". potentially a "waste" but maybe not really
	// seeing as the UDP connection is connectionless

	// I think one trouble with this is that all clients have to set up a listener for every peer in case that peer wants
	// to talk to them. they can lazily only connect to an external peer when/if they want to talk to that peer though.
	// no actually i think they can just have one listener that accepts data from all peers. and then they connect to external
	// peers if/when they want.

	// another trouble is i think i need to use the udp port that i've connected to the rollodex on as the port to accept connections on.

	// so I think
	// rollodex runs a server on a specified port/ip
	// clients create UDP socket that connects to the rollodex
	// clients create DTLS client that uses that socket
	// clients also create a DTLS server that uses that socket -- does that work?

	// so the server is on samlewis.me:12345
	// client starts up, connects to samlewis.me:12345 and the socket is locally bound to samhome.local:10000
	// so then maybe samhome.local:10001 can listen for dtls connections
	//
	// so then bobs client starts up, connects to samlewis.me:12345 and socket is locally bound to bobhome.local:20000
	// bob wants to talk to sam so connects to samhome.local:10001 using

	// is there a way that I can manage the DTLS sessions by choosing what to do with the data depending on where it comes from?
	// ie if it comes from the rollodex then feed the data to that connection?
}

// Simple comparison to see if this member should be the DTLS server or if the remote member should be
func (c *MeshMember) AmServer(other netaddr.IPPort) bool {
	ipCompare := c.myOutsideIP.IP.Compare(other.IP)

	switch ipCompare {
	case -1:
		return false
	case 0:
		if c.myOutsideIP.Port > other.Port {
			return true
		} else if c.myOutsideIP.Port < other.Port {
			return false
		} else {
			panic("Remote IPPort == Local IPPort")
		}
	case 1:
		return true
	default:
		panic("Unexpected comparison result")
	}
}

func (p *Peer) PeerServe() {
	// send my internal IP address until

	//dtlsConn.ConnectionState().IdentityHint to send the internal IP address?
	// p.tunIP.String()

	// // this comparison should be on the external IP addresses
	// // as we don't know the internal ones yet
	// myIpPort, _ := netaddr.ParseIPPort("127.0.0.1:3333")
	// remoteIpPort, _ := netaddr.ParseIPPort("127.0.0.1:3343")

	// ipPort, err := netaddr.ParseIPPort(p.tunIP.String())

	// ipPort.IP
}

func NewMeshMember(rollodexIp string, rollodexPort int, tun *tun.Tun) *MeshMember {
	member := MeshMember{rollodexIp: rollodexIp, tun: tun}
	member.quit = make(chan bool)

	// config := &dtls.Config{
	// 	PSK: func(hint []byte) ([]byte, error) {
	// 		fmt.Printf("Client's hint: %s \n", hint)
	// 		return []byte{0xAB, 0xC1, 0x23}, nil
	// 	},
	// 	PSKIdentityHint:      []byte("Pion DTLS Client"),
	// 	CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8},
	// 	ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	// 	// Create timeout context for accepted connection.
	// 	ConnectContextMaker: func() (context.Context, func()) {
	// 		return context.WithTimeout(ctx, 30*time.Second)
	// 	},
	// }

	myAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0")}
	rollodexAddr := &net.UDPAddr{IP: net.ParseIP(rollodexIp), Port: rollodexPort}

	listener, err := udp.Listen("udp", myAddr)

	if err != nil {
		log.Fatalln(err)
	}

	member.listener = listener.(*udp.Listener)
	rolloConn, err := member.listener.CreateConn(rollodexAddr)

	member.rolloConn = rolloConn
	member.peers = make(map[netaddr.IPPort]*Peer)

	return &member
}

func (c *MeshMember) readLoop(conn net.Conn) {
	b := make([]byte, bufSize)
	for {
		n, err := conn.Read(b)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Got message: %s\n", string(b[:n]))
	}
}

// Chat starts the stdin readloop to dispatch messages to the hub
func (c *MeshMember) sendLoop(conn net.Conn) {
	counter := 0
	for {
		msg := fmt.Sprintf("Hello %d from %v", counter, c.myOutsideIP)
		counter += 1
		n, err := conn.Write([]byte(msg))

		fmt.Printf("Wrote %d\n", n)

		if err != nil {
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *MeshMember) connectToNewPeer(address netaddr.IPPort) error {
	config := &dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			fmt.Printf("Server's hint: %s \n", hint)
			return []byte{0xAB, 0xC1, 0x23}, nil
		},
		PSKIdentityHint:      []byte(c.myOutsideIP.String()),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}

	//ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	//defer cancel()
	conn, err := c.listener.CreateConn(address.UDPAddr())

	if err != nil {
		return err
	}

	var dtlsConn *dtls.Conn

	if c.AmServer(address) {
		// the other dude will connect to us
		dtlsConn, err = dtls.Server(conn, config)
	} else {
		dtlsConn, err = dtls.Client(conn, config)
	}

	if err != nil {
		return err
	}

	remoteTunIp := string(dtlsConn.ConnectionState().IdentityHint)

	tunIP, err := netaddr.ParseIPPort(remoteTunIp)

	if err != nil {
		return err
	}

	peer := &Peer{remoteIP: address, tunIP: tunIP, conn: dtlsConn, lastContacted: time.Now()}
	c.peers[address] = peer

	fmt.Printf("Successfully connected to new peer %v\n", peer)

	go c.readLoop(dtlsConn)
	go c.sendLoop(dtlsConn)

	return nil
}

func (c *MeshMember) newAddresses(addreses []netaddr.IPPort) {
	for _, address := range addreses {
		_, ok := c.peers[address]

		if ok {
			// we already know of this peer
			fmt.Printf("Already connected to %v\n", address)
			continue
		}

		if address == c.myOutsideIP {
			// don't connect to myself
			continue
		}

		fmt.Printf("Going to try to connect to %v\n", address)
		fmt.Printf("Currently connected to %v\n", c.peers)

		if err := c.connectToNewPeer(address); err != nil {
			fmt.Println("Could not connect to ", address, err)
			continue
		}
		// we need to connect to the other dude

		// if we don't know about it, see if we should be the server or client
		// then connect to it if we need to
		// on connection from another server, need to check its hint to see the internal IP
		// then add to the map

		// when we get data from our tun look at the map to see where we should forward the data onto
		// is it all really this simple? maybe im a genius
	}
}

func (c *MeshMember) RolloReadLoop() {
	buf := make([]byte, 65535)

	for {
		n, err := c.rolloConn.Read(buf)

		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			println("Got error")
			continue
		}

		var members NetworkMap

		if err := json.Unmarshal(buf[:n], &members); err != nil {
			println("Error unmarshalling ", err.Error())
			continue
		}

		c.myOutsideIP = members.Addresses[members.YourIndex]

		fmt.Printf("%+v\n", members)

		c.newAddresses(members.Addresses)
	}
}

func (c *MeshMember) RolloSendLoop() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			break
		}

		heartbeat := HeartbeatMessage{NetworkName: "samsNetwork"}
		b, err := json.Marshal(heartbeat)
		if err != nil {
			panic(err)
		}

		println(string(b))
		_, err = c.rolloConn.Write(b)

		if err != nil {
			panic(err)
		}
	}
}
