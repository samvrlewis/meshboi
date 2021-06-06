package meshboi

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestRolodex(t *testing.T) {
	conn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 33333})
	rollo, err := NewRolodex(conn, 1*time.Second, 5*time.Second)

	if err != nil {
		t.FailNow()
	}

	go rollo.Run()

	client, err := net.Dial("udp", "127.0.0.1:33333")

	client.Write([]byte(`{"networkName": "test"}`))

	time.Sleep(2 * time.Second)

	buf := make([]byte, 1000)

	client.Read(buf)

	if !bytes.Contains(buf, []byte("127.0.0.1")) {
		t.Fatalf("Didn't get back expected IP")
	}
}
