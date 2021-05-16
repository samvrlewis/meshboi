package meshboi

import (
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

func (pc *PeerConnector) connectToNewPeer(address netaddr.IPPort) error {
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

func (pc *PeerConnector) newAddresses(addreses []netaddr.IPPort) {
	for _, address := range addreses {
		_, ok := pc.store.GetByOutsideIpPort(address)

		if ok {
			// we already know of this peer
			log.Info("Already connected to ", address)
			continue
		}

		if address == pc.myOutsideAddr {
			// don't connect to myself
			continue
		}

		if pc.AmServer(address) {
			// the other dude will connect to us
			continue
		}

		log.Info("Going to try to connect to ", address)

		if err := pc.connectToNewPeer(address); err != nil {
			log.Warn("Could not connect to ", address, err)
			continue
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
