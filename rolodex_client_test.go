package meshboi

import (
	"bytes"
	"net"
	"testing"
	"time"

	"inet.af/netaddr"
)

func TestNetworkCallback(t *testing.T) {
	var nmap NetworkMap

	called := 0
	callback := func(member NetworkMap) {
		called += 1
		nmap = member
	}
	client, server := net.Pipe()
	rolloClient := NewRolodexClient("testNet", client, time.Second, callback)

	go rolloClient.Run()
	defer rolloClient.Stop()

	server.Write([]byte(`{ "addresses": ["192.168.4.1:2000"], "your_index": 0 }`))

	time.Sleep(time.Millisecond)
	if len(nmap.Addresses) != 1 {
		t.Fatalf("expected 1 address but got %v", len(nmap.Addresses))
	}

	if called != 1 {
		t.Fatalf("Called more than once")
	}

	if nmap.Addresses[0] != netaddr.MustParseIPPort("192.168.4.1:2000") {
		t.Fatalf("Wrong ip address back")
	}

	if nmap.YourIndex != 0 {
		t.Fatalf("Wrong index back")
	}
}

func TestClientSendsHeartBeat(t *testing.T) {
	callback := func(member NetworkMap) {
	}
	client, server := net.Pipe()
	rolloClient := NewRolodexClient("testNet", client, time.Millisecond, callback)
	go rolloClient.Run()
	defer rolloClient.Stop()

	time.Sleep(time.Millisecond)

	b := make([]byte, 1000)
	n, _ := server.Read(b)

	if !bytes.Contains(b[:n], []byte("testNet")) {
		t.Fatalf("Didn't contain the network name %v", string(b[:n]))
	}
}
