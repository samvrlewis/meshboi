package meshboi

import (
	"net"
	"testing"

	"inet.af/netaddr"
)

type testDialer struct {
	dialer net.Dialer
	dialed chan (net.Addr)
}

func (t *testDialer) Dial(raddr net.Addr) (net.Conn, error) {
	t.dialed <- raddr
	c, _ := net.Pipe()
	return c, nil
}

func TestPeerConnector(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:3333")

	if err != nil {
		t.Fatalf("%v", err)
	}

	td := testDialer{dialed: make(chan net.Addr)}
	store := NewPeerConnStore()
	client, _ := net.Pipe()

	pc := NewPeerConnector(netaddr.MustParseIP("192.168.4.1"), listener, &td, store, client)

	go pc.ListenForPeers()

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
