package meshboi

import (
	"bytes"
	"net"
	"testing"

	"inet.af/netaddr"
)

type FakeTun struct {
	bytes.Buffer
}

func (f *FakeTun) Close() error {
	return nil
}

func NewFakePeerConn(inside string, outside string) *PeerConn {
	insideIP := netaddr.MustParseIP(inside)
	outsideIP := netaddr.MustParseIPPort(outside)
	_, client := net.Pipe()
	p := NewPeerConn(insideIP, outsideIP, client, &FakeTun{})
	return &p
}

type test struct {
	insideIP  string
	outsideIP string
}

var tests = []test{
	{insideIP: "192.168.4.1", outsideIP: "192.168.44.1:5000"},
	{insideIP: "10.0.0.1", outsideIP: "1.1.1.1:2334"},
	{insideIP: "2.2.2.2", outsideIP: "3.4.3.3:2"},
}

func TestGetByIP(t *testing.T) {
	store := NewPeerConnStore()

	for _, tc := range tests {
		pc := NewFakePeerConn(tc.insideIP, tc.outsideIP)
		store.Add(pc)

		retrievedPeerConn, ok := store.GetByInsideIp(netaddr.MustParseIP(tc.insideIP))

		if !ok {
			t.Errorf("Couldn't find peer conn by inside IP")
		}

		if retrievedPeerConn != pc {
			t.Errorf("Wrong peer conn returned for inside IP")
		}

		retrievedPeerConn, ok = store.GetByOutsideIpPort(netaddr.MustParseIPPort(tc.outsideIP))

		if !ok {
			t.Errorf("Couldn't find peer conn by outside IP")
		}

		if retrievedPeerConn != pc {
			t.Errorf("Wrong peer conn returned for outside IP")
		}
	}
}

func TestGetNotExisting(t *testing.T) {
	store := NewPeerConnStore()

	_, ok := store.GetByInsideIp(netaddr.MustParseIP("192.168.1.1"))

	if ok {
		t.Errorf("Shouldn't have gotten a peer conn back")
	}

	_, ok = store.GetByOutsideIpPort(netaddr.MustParseIPPort("192.168.1.1:5000"))

	if ok {
		t.Errorf("Shouldn't have gotten a peer conn back")
	}
}

func TestDeleteByIP(t *testing.T) {
	store := NewPeerConnStore()

	pc := NewFakePeerConn(tests[0].insideIP, tests[0].outsideIP)
	store.Add(pc)

	ok := store.RemoveByOutsideIPPort(netaddr.MustParseIPPort(tests[0].outsideIP))

	if !ok {
		t.Errorf("Could not remove peer conn")
	}

	_, ok = store.GetByInsideIp(netaddr.MustParseIP(tests[0].insideIP))

	if ok {
		t.Errorf("Found deleted peer")
	}

	_, ok = store.GetByOutsideIpPort(netaddr.MustParseIPPort(tests[0].outsideIP))

	if ok {
		t.Errorf("Found deleted peer")
	}

}

func TestDeleteNonExistentIP(t *testing.T) {
	store := NewPeerConnStore()

	ok := store.RemoveByOutsideIPPort(netaddr.MustParseIPPort(tests[0].outsideIP))

	if ok {
		t.Errorf("Deleting a non existing IP Port shouldn't work")
	}
}
