package meshboi

import (
	"net"
	"reflect"
	"testing"

	"inet.af/netaddr"
)

// Tests that queued data goes to the peer
func TestSendData(t *testing.T) {
	client, server := net.Pipe()
	tun := FakeTun{}
	conn := NewPeerConn(netaddr.MustParseIP("192.168.5.1"), netaddr.MustParseIPPort("192.168.33.1:5000"), client, &tun)
	msg := []byte("hello this is some data")

	go conn.sendLoop()

	conn.QueueData(msg)

	b := make([]byte, 1000)
	n, _ := server.Read(b)

	if !reflect.DeepEqual(b[:n], msg) {
		t.Fatalf("Didn't read expected data")
	}
}

// Tests that data received externally from the peer goes to the tun
func TestReceiveData(t *testing.T) {
	client, server := net.Pipe()
	tunClient, tunServer := net.Pipe()
	conn := NewPeerConn(netaddr.MustParseIP("192.168.5.1"), netaddr.MustParseIPPort("192.168.33.1:5000"), client, tunClient)
	go conn.readLoop()

	msg := []byte("hello this is some data")
	server.Write(msg)
	b := make([]byte, 1000)
	n, _ := tunServer.Read(b)

	if !reflect.DeepEqual(b[:n], msg) {
		t.Fatalf("Didn't read expected data %v %v", b[:n], msg)
	}
}
