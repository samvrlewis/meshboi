package meshboi

import (
	"encoding/json"
	"net"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/pkg/protocol"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
	"github.com/samvrlewis/meshboi/tun"
	"github.com/samvrlewis/udp"
	"golang.org/x/net/ipv4"
	"inet.af/netaddr"

	log "github.com/sirupsen/logrus"
)

const bufSize = 8192

const (
	MTU_SIZE = 1300
)

type MeshMember struct {
	rollodexIp  string
	tun         *tun.Tun
	quit        chan bool
	rolloConn   net.Conn
	myOutsideIP netaddr.IPPort
	myInsideIP  netaddr.IP

	// This manages and multiplexes a set of udp "connections" that are all on a
	// connected to a single local port.
	listener *udp.Listener

	peerStore *PeerStore

	startedListening bool

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

func NewMeshMember(rollodexIp string, rollodexPort int, tun *tun.Tun, myInsideIP netaddr.IP) *MeshMember {
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

	lc := udp.ListenConfig{
		AcceptFilter: func(packet []byte) bool {
			pkts, err := recordlayer.UnpackDatagram(packet)
			if err != nil || len(pkts) < 1 {
				return false
			}
			h := &recordlayer.Header{}
			if err := h.Unmarshal(pkts[0]); err != nil {
				return false
			}
			return h.ContentType == protocol.ContentTypeHandshake
		},
	}

	listener, err := lc.Listen("udp", myAddr)

	if err != nil {
		log.Error("Error creating listener ", err)
	}

	member.listener = listener.(*udp.Listener)
	rolloConn, err := member.listener.CreateConn(rollodexAddr)

	member.rolloConn = rolloConn
	member.peerStore = NewPeerStore()
	member.startedListening = false
	member.myInsideIP = myInsideIP

	go member.tunReadLoop()

	return &member
}

func (c *MeshMember) listen() {
	c.startedListening = true

	config := &dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			return []byte{0xAB, 0xC1, 0x23}, nil
		},
		PSKIdentityHint:      []byte(c.myInsideIP.String()),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}

	for {
		conn, err := c.listener.Accept()

		if err != nil {
			log.Warn("Error accepting: ", err)
			continue
		}

		dtlsConn, err := dtls.Server(conn, config)

		if err != nil {
			log.Warn("Error starting dtls connection: ", err)
			conn.Close()
			continue
		}

		remoteTunIp := string(dtlsConn.ConnectionState().IdentityHint)

		tunIP, err := netaddr.ParseIP(remoteTunIp)

		if err != nil {
			log.Warn("Error parsing tunIP from hint: ", err)
			dtlsConn.Close()
			continue
		}

		log.Info("Succesfully accepted connection from ", dtlsConn.RemoteAddr())

		// todo: Is it necessary to remember what the remote ip is here?
		peer := &Peer{tunIP: tunIP, conn: dtlsConn, lastContacted: time.Now(), outgoing: make(chan []byte), member: c}

		// todo: This is wrong but I can't figure out how to get the actual tunIP
		c.peerStore.Add(peer)

		go peer.readLoop()
		go peer.sendLoop()
	}
}

func (c *MeshMember) connectToNewPeer(address netaddr.IPPort) error {
	config := &dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			return []byte{0xAB, 0xC1, 0x23}, nil
		},
		PSKIdentityHint:      []byte(c.myInsideIP.String()),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}

	//ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	//defer cancel()
	conn, err := c.listener.CreateConn(address.UDPAddr())

	if err != nil {
		return err
	}

	dtlsConn, err := dtls.Client(conn, config)

	if err != nil {
		conn.Close()
		return err
	}

	remoteTunIp := string(dtlsConn.ConnectionState().IdentityHint)

	tunIP, err := netaddr.ParseIP(remoteTunIp)

	if err != nil {
		dtlsConn.Close()
		return err
	}

	peer := &Peer{remoteIP: address, tunIP: tunIP, conn: dtlsConn, lastContacted: time.Now(), member: c, outgoing: make(chan []byte)}
	c.peerStore.Add(peer)

	log.Info("Successfully connected to new peer ", peer)

	go peer.readLoop()
	go peer.sendLoop()

	return nil
}

func (c *MeshMember) newAddresses(addreses []netaddr.IPPort) {
	for _, address := range addreses {
		_, ok := c.peerStore.GetByOutsideIpPort(address)

		if ok {
			// we already know of this peer
			log.Info("Already connected to ", address)
			continue
		}

		if address == c.myOutsideIP {
			// don't connect to myself
			continue
		}

		if c.AmServer(address) {
			// the other dude will connect to us
			continue
		}

		log.Info("Going to try to connect to ", address)

		if err := c.connectToNewPeer(address); err != nil {
			log.Warn("Could not connect to ", address, err)
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
			log.Warn("Temporary error reading from rolloConn: ", nerr)
			continue
		}

		var members NetworkMap

		if err := json.Unmarshal(buf[:n], &members); err != nil {
			log.Error("Error unmarshalling incoming message: ", err.Error())
			continue
		}

		c.myOutsideIP = members.Addresses[members.YourIndex]

		if !c.startedListening {
			go c.listen()
		}

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
			log.Fatalln("Error marshalling JSON heartbeat message: ", err)
		}

		_, err = c.rolloConn.Write(b)

		if err != nil {
			log.Error("Error sending heartbeat over the rollo conn: ", err)
		}
	}
}

func (c *MeshMember) sendToPeer(vpnIP netaddr.IP, data []byte) {
	peer, ok := c.peerStore.GetByInsideIp(vpnIP)

	if !ok {
		log.Info("Dropping data destined for ", vpnIP)
		return
	}

	peer.addData(data)
}

func (c *MeshMember) tunReadLoop() {
	packet := make([]byte, MTU_SIZE)
	for {
		// Check if we should quit before continuing
		select {
		case <-c.quit:
			return
		default:
			break
		}

		n, err := c.tun.Read(packet)
		if err != nil {
			log.Error("Error reading from tun device: ", err)
			continue
		}

		header, err := ipv4.ParseHeader(packet[:n])

		if err != nil {
			log.Error("Error parsing ipv4 header of tun packet: ", err)
			continue
		}

		fromIp, ok := netaddr.FromStdIP(header.Dst)

		if !ok {
			log.Error("Error converting to netaddr IP")
			continue
		}

		c.sendToPeer(fromIp, packet[:n])
	}
}
