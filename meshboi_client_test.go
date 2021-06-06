package meshboi

import (
	"net"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/ipv4"
	"inet.af/netaddr"
)

// Simple test to test data flows from one client to another
func TestTwoClients(t *testing.T) {

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345})

	if err != nil {
		t.Error("Couldn't make UDP listener: ", err)
	}

	rolodex, err := NewRolodex(conn, time.Second, time.Minute)

	if err != nil {
		t.Error("Couldn't make rolodex: ", err)
	}

	go rolodex.Run()

	// use these as fake tuns
	// the incoming is the tun to the outside world (ie what other applications would write to)
	// and the outgoing is what meshboi reads and writes to
	tunIncoming1, tunOutgoing1 := net.Pipe()
	tunIncoming2, tunOutgoing2 := net.Pipe()

	client1, err := NewMeshBoiClient(tunOutgoing1, netaddr.MustParseIPPrefix("192.168.52.1/24"), netaddr.MustParseIP("127.0.0.1"), 12345, "testNetwork", []byte("testpassword"))

	if err != nil {
		t.Error("Error making mesh client ", err)
	}

	client2, err := NewMeshBoiClient(tunOutgoing2, netaddr.MustParseIPPrefix("192.168.52.2/24"), netaddr.MustParseIP("127.0.0.1"), 12345, "testNetwork", []byte("testpassword"))

	if err != nil {
		t.Error("Error making mesh client ", err)
	}

	go client1.Run()
	defer client1.Stop()

	go client2.Run()
	defer client2.Stop()

	b := []byte("hello how are you?")
	h := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TotalLen: ipv4.HeaderLen + len(b),
		ID:       55555,
		Protocol: 1,
		Dst:      net.ParseIP("192.168.52.2"),
	}

	header, err := h.Marshal()

	if err != nil {
		t.Error("Error marshalling header ", err)
	}

	sentMsg := append(header, b...)

	// The connection between the peers takes at least 1 second to create
	//
	// todo: It would be much nicer if we could get the client to inform us when
	// the connection has been made so we could wait on a condition var or channel
	// instead of needing to sleep here
	time.Sleep(2 * time.Second)
	tunIncoming1.Write(sentMsg)

	var rxedMsg []byte

	rxedMsg = make([]byte, 4000)

	n, err := tunIncoming2.Read(rxedMsg)

	if err != nil {
		t.Error("Error reading from tun ", err)
	}

	if !reflect.DeepEqual(rxedMsg[:n], sentMsg) {
		t.Errorf("Didn't read expected data %v %v", rxedMsg[:n], sentMsg)
	}
}
