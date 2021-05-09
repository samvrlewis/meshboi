package meshboi

import (
	"net"

	"inet.af/netaddr"

	"github.com/pion/dtls/v2"
	"github.com/samvrlewis/meshboi/tun"

	log "github.com/sirupsen/logrus"
)

type PeerConnector struct {
	store         *PeerConnStore
	listener      net.Listener
	dialer        PeerDialer
	myOutsideAddr netaddr.IPPort
	myInsideIP    netaddr.IP
	tun           *tun.Tun
}

func (pc *PeerConnector) GetDtlsConfig() *dtls.Config {
	return &dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			return []byte{0xAB, 0xC1, 0x23}, nil
		},
		PSKIdentityHint:      []byte(pc.myInsideIP.String()),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}
}

// Simple comparison to see if this member should be the DTLS server or if the remote member should be
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

func NewPeerConnector(insideIp netaddr.IP, peerListener net.Listener, peerDialer PeerDialer, store *PeerConnStore, tun *tun.Tun) PeerConnector {
	return PeerConnector{
		listener:   peerListener,
		dialer:     peerDialer,
		store:      store,
		myInsideIP: insideIp,
		tun:        tun,
	}
}

func (pc *PeerConnector) OnNetworkMapUpdate(network NetworkMap) {
	pc.myOutsideAddr = network.Addresses[network.YourIndex]
	pc.newAddresses(network.Addresses)
}

func (pc *PeerConnector) connectToNewPeer(address netaddr.IPPort) error {
	conn, err := pc.dialer.Dial(address.UDPAddr())

	if err != nil {
		return err
	}

	dtlsConn, err := dtls.Client(conn, pc.GetDtlsConfig())

	if err != nil {
		conn.Close()
		return err
	}

	return pc.OnNewPeerConnection(dtlsConn)
}

func (pc *PeerConnector) OnNewPeerConnection(conn *dtls.Conn) error {
	peerIpString := string(conn.ConnectionState().IdentityHint)
	peerVpnIP, err := netaddr.ParseIP(peerIpString)

	if err != nil {
		log.Warn("Error parsing tunIP from hint: ", err)
		conn.Close()
		return err
	}

	outsideAddr, err := netaddr.ParseIPPort(conn.RemoteAddr().String())

	if err != nil {
		conn.Close()
		return err
	}

	log.Info("Succesfully accepted connection from ", conn.RemoteAddr())

	peer := NewPeerConn(peerVpnIP, outsideAddr, conn, pc.tun)

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
		conn, err := pc.listener.Accept()

		if err != nil {
			log.Warn("Error accepting: ", err)
			continue
		}

		dtlsConn, err := dtls.Server(conn, pc.GetDtlsConfig())

		if err != nil {
			log.Warn("Error starting dtls connection: ", err)
			conn.Close()
			continue
		}

		pc.OnNewPeerConnection(dtlsConn)
	}
}
