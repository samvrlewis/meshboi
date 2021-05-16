package meshboi

import (
	"net"

	"github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/pkg/protocol"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
	"github.com/samvrlewis/udp"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

// VpnListenerDialer allows for:
// 	- Dialing connections to other members in the VPN Mesh
//	- Accepting connections to other members in the VPN Mesh
// 	- Dialing connections to non VPN Mesh members
type VpnMeshListenerDialer interface {
	// Returns the connection and the VPN IP address on the other side
	AcceptMesh() (MeshConn, error)
	// Returns the connection and the VPN IP address on the other side
	DialMesh(raddr net.Addr) (MeshConn, error)
	Dial(raddr net.Addr) (net.Conn, error)
}

type MeshConn interface {
	net.Conn
	RemoteMeshAddr() netaddr.IP
}

type meshConn struct {
	net.Conn
	remoteMeshAddr netaddr.IP
}

func (m *meshConn) RemoteMeshAddr() netaddr.IP {
	return m.remoteMeshAddr
}

// MultiplexedDTLSConn represents a conn that can be used to listen for new incoming DTLS connections
// and also dial new UDP connections (both DTLS and non-DTLS) from the same udp address
type MultiplexedDTLSConn struct {
	listener *udp.Listener
	config   *dtls.Config
}

func NewMultiplexedDTLSConn(laddr *net.UDPAddr, config *dtls.Config) (*MultiplexedDTLSConn, error) {
	// Set a listen config so that we only accept incoming connections that are DTLS connections
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

	listener, err := lc.Listen("udp", laddr)

	if err != nil {
		return nil, err
	}

	return &MultiplexedDTLSConn{
		listener: listener.(*udp.Listener),
		config:   config,
	}, nil
}

func (mc *MultiplexedDTLSConn) startDtlsConn(conn net.Conn, isServer bool) (MeshConn, error) {
	var dtlsConn *dtls.Conn
	var err error

	if isServer {
		dtlsConn, err = dtls.Server(conn, mc.config)
	} else {
		dtlsConn, err = dtls.Client(conn, mc.config)
	}

	if err != nil {
		log.Warn("Error starting dtls connection: ", err)
		conn.Close()
		return nil, err
	}

	peerIpString := string(dtlsConn.ConnectionState().IdentityHint)
	peerVpnIP, err := netaddr.ParseIP(peerIpString)

	if err != nil {
		log.Warn("Couldn't parse peers vpn IP")
		return nil, err
	}

	return &meshConn{Conn: dtlsConn,
		remoteMeshAddr: peerVpnIP,
	}, nil
}

func (mc *MultiplexedDTLSConn) AcceptMesh() (MeshConn, error) {
	conn, err := mc.listener.Accept()

	if err != nil {
		return nil, err
	}

	return mc.startDtlsConn(conn, true)

}

func (mc *MultiplexedDTLSConn) DialMesh(raddr net.Addr) (MeshConn, error) {
	conn, err := mc.listener.Dial(raddr)

	if err != nil {
		return nil, err
	}

	return mc.startDtlsConn(conn, false)
}

func (mc *MultiplexedDTLSConn) Dial(raddr net.Addr) (net.Conn, error) {
	return mc.listener.Dial(raddr)
}
