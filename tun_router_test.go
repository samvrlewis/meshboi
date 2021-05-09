package meshboi

import (
	"net"
	"reflect"
	"testing"

	"golang.org/x/net/ipv4"
	"inet.af/netaddr"
)

func TestRouter(t *testing.T) {
	store := NewPeerConnStore()
	tunClient, tunServer := net.Pipe()
	tr := NewTunRouter(tunClient, store)
	go tr.Run()
	defer tr.Stop()

	peerClient, peerServer := net.Pipe()

	peer := NewPeerConn(netaddr.MustParseIP("192.168.4.3"), netaddr.MustParseIPPort("192.152.12.2:2222"), peerClient, tunClient)
	go peer.readLoop()
	go peer.sendLoop()
	store.Add(&peer)

	hdr := ipv4.Header{
		Src:     net.ParseIP("192.168.4.2"),
		Dst:     net.ParseIP("192.168.4.3"),
		Len:     20,
		Version: 4,
	}

	hdrBytes, _ := hdr.Marshal()

	msg := append(hdrBytes[:], []byte("hello")...)

	tunServer.Write(msg)

	readBytes := make([]byte, 1000)

	n, _ := peerServer.Read(readBytes)

	if !reflect.DeepEqual(readBytes[:n], msg) {
		t.Errorf("Messages not equal")
	}
}
