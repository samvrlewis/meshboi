package meshboi

import (
	"net"

	"github.com/pion/dtls/v2/pkg/protocol"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
	"github.com/samvrlewis/udp"
)

type PeerDialer interface {
	Dial(raddr net.Addr) (net.Conn, error)
}

// MultiplexedDTLSConn represents a conn that can be used to listen for new incoming DTLS connections
// and also dial new UDP connections (both DTLS and non-DTLS) from the same udp address
type MultiplexedDTLSConn struct {
	listener *udp.Listener
}

func NewMultiplexedDTLSConn(laddr *net.UDPAddr) (*MultiplexedDTLSConn, error) {
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
	}, nil
}

func (mc *MultiplexedDTLSConn) GetListener() net.Listener {
	return mc.listener
}

func (mc *MultiplexedDTLSConn) GetDialer() PeerDialer {
	return mc.listener
}
