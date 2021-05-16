package meshboi

import (
	"net"
	"testing"

	"inet.af/netaddr"
)

type testListenerDialer struct {
	dialer net.Dialer
	dialed chan (net.Addr)
}

func (t testListenerDialer) DialMesh(raddr net.Addr) (MeshConn, error) {
	t.dialed <- raddr
	c, _ := net.Pipe()
	ip := netaddr.MustParseIP("192.168.1.1")
	return &meshConn{
		Conn:           c,
		remoteMeshAddr: ip,
	}, nil
}

func (t testListenerDialer) AcceptMesh() (MeshConn, error) {
	return nil, nil
}

func (t testListenerDialer) Dial(raddr net.Addr) (net.Conn, error) {
	return nil, nil
}

func TestPeerConnector(t *testing.T) {
	td := testListenerDialer{dialed: make(chan net.Addr)}
	store := NewPeerConnStore()
	client, _ := net.Pipe()

	pc := NewPeerConnector(td, store, client)

	nm := NetworkMap{
		Addresses: []netaddr.IPPort{netaddr.MustParseIPPort("192.168.33.1:3000"),
			netaddr.MustParseIPPort("192.168.33.2:4000")},
		YourIndex: 0,
	}

	go pc.OnNetworkMapUpdate(nm)

	dialed := <-td.dialed

	if dialed.String() != "192.168.33.2:4000" {
		t.Fatalf("Dialed wrong address")
	}
}
