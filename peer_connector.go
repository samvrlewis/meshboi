package meshboi

import (
	"net"
	"time"

	"inet.af/netaddr"

	log "github.com/sirupsen/logrus"
)

type PeerConnector struct {
	store          *PeerConnStore
	listenerDialer VpnMeshListenerDialer
	myOutsideAddr  netaddr.IPPort
	tun            TunConn
}

// Simple comparison to see if this member should be the server or if the remote member should be
func (pc *PeerConnector) AmServer(other netaddr.IPPort) bool {
	ipCompare := pc.myOutsideAddr.IP.Compare(other.IP)

	switch ipCompare {
	case -1:
		return false
	case 0:
		if pc.myOutsideAddr.Port > other.Port {
			return true
		} else if pc.myOutsideAddr.Port < other.Port {
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

func NewPeerConnector(listenerDialer VpnMeshListenerDialer, store *PeerConnStore, tun TunConn) PeerConnector {
	return PeerConnector{
		listenerDialer: listenerDialer,
		store:          store,
		tun:            tun,
	}
}

func (pc *PeerConnector) OnNetworkMapUpdate(network NetworkMap) {
	pc.myOutsideAddr = network.Addresses[network.YourIndex]
	pc.newAddresses(network.Addresses)
}

func (pc *PeerConnector) readAllFromAddr(address net.Addr, timeout time.Duration) error {
	conn, err := pc.listenerDialer.Dial(address)

	if err != nil {
		log.Warn("Error connecting to peer unencrypted ", err)
		return err
	}

	defer conn.Close()

	buf := make([]byte, 1024)

	conn.SetReadDeadline(time.Now().Add(timeout))
	n, err := conn.Read(buf)

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Println("Read timeout:", err)
		} else {
			log.Println("Read error:", err)
			return err
		}
	} else {
		log.Info("Read: ", string(buf[:n]))
	}

	return nil
}

func (pc *PeerConnector) connectToNewPeer(address netaddr.IPPort) error {
	// We are going to initiate a dTLS connection to the other mesh member
	// however, for it to open a hole in its firewall it has sent us an initial
	// message if we have our own firewall then this message will likely not be
	// received but if it does get through it can wait here to receive it before
	// continuing
	err := pc.readAllFromAddr(address.UDPAddr(), time.Second)

	if err != nil {
		return err
	}

	conn, err := pc.listenerDialer.DialMesh(address.UDPAddr())

	if err != nil {
		return err
	}

	return pc.OnNewPeerConnection(conn)
}

func (pc *PeerConnector) OnNewPeerConnection(conn MeshConn) error {
	outsideAddr, err := netaddr.ParseIPPort(conn.RemoteAddr().String())

	if err != nil {
		conn.Close()
		return err
	}

	log.Info("Succesfully accepted connection from ", conn.RemoteAddr())

	peer := NewPeerConn(conn.RemoteMeshAddr(), outsideAddr, conn, pc.tun)

	pc.store.Add(&peer)

	go peer.readLoop()
	go peer.sendLoop()

	return nil
}

func (pc *PeerConnector) openFirewallToPeer(addr net.Addr) error {
	conn, err := pc.listenerDialer.Dial(addr)
	defer conn.Close()

	if err != nil {
		log.Warn("Error connecting to peer unencrypted ", err)
		return err
	}

	// It doesn't really matter what is sent here - the important part is
	// something is sent. We're effectively telling any and all (stateful)
	// firewalls on our path to the peer to allow any future traffic that has
	// originated from that peer
	_, err = conn.Write([]byte("hello"))

	if err != nil {
		log.Warn("Error writing to peer unencrypted ", err)
		return err
	}

	return nil
}

func (pc *PeerConnector) newAddresses(addreses []netaddr.IPPort) {
	for _, address := range addreses {
		_, ok := pc.store.GetByOutsideIpPort(address)

		if ok {
			// we already know of this peer
			continue
		}

		if address == pc.myOutsideAddr {
			// don't connect to myself
			continue
		}

		if pc.AmServer(address) {
			peer := NewPeerConn(netaddr.IP{}, address, nil, pc.tun)
			pc.store.Add(&peer)

			// As the peer will initiate connection to our dTLS server we first
			// need to make sure our firewall(s) are open to allow the peer to
			// contact us
			err := pc.openFirewallToPeer(address.UDPAddr())

			if err != nil {
				log.Warn("Error opening firewall: ", err)

				// Remove the peer so we can try again later
				pc.store.RemoveByOutsideIPPort(address)
			}
		} else {
			log.Info("Going to try to connect to ", address)

			if err := pc.connectToNewPeer(address); err != nil {
				log.Warn("Could not connect to ", address, err)
				continue
			}
		}
	}
}

func (pc *PeerConnector) ListenForPeers() {
	for {
		conn, err := pc.listenerDialer.AcceptMesh()

		if err != nil {
			log.Warn("Error accepting: ", err)
			continue
		}

		pc.OnNewPeerConnection(conn)
	}
}

func (pc *PeerConnector) Stop() {
	// todo: Properly shut down all the spawned peers
}
