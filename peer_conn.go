package meshboi

import (
	"net"
	"time"

	"github.com/samvrlewis/meshboi/tun"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

const bufSize = 65535

// Represents a connection to a peer
type PeerConn struct {
	// The IP address within the VPN
	insideIP netaddr.IP

	// The IP address over the internet
	outsideAddr netaddr.IPPort

	// Time of last contact
	lastContacted time.Time

	// the connection to the peer
	conn     net.Conn
	outgoing chan []byte
	tun      *tun.Tun
}

func NewPeerConn(insideIP netaddr.IP, outsideAddr netaddr.IPPort, conn net.Conn, tun *tun.Tun) PeerConn {
	return PeerConn{
		insideIP:      insideIP, // maybe these dont need to be inside the peer. could just be in the peer store
		outsideAddr:   outsideAddr,
		conn:          conn,
		tun:           tun,
		lastContacted: time.Now(),
		outgoing:      make(chan []byte),
	}
}

func (p *PeerConn) QueueData(data []byte) {
	p.outgoing <- data
}

func (p *PeerConn) readLoop() {
	b := make([]byte, bufSize)
	for {
		n, err := p.conn.Read(b)
		if err != nil {
			panic(err)
		}

		p.lastContacted = time.Now()

		log.Info("Got message: ", b[:n])

		written, err := p.tun.Write(b[:n])

		if err != nil {
			panic(err)
		}

		if written != n {
			log.Warn("Not all data written to tun")
		}
	}
}

// Chat starts the stdin readloop to dispatch messages to the hub
func (p *PeerConn) sendLoop() {
	for {
		data := <-p.outgoing

		log.Info("Going to send to peer")
		log.Info("Sending message: ", data)
		n, err := p.conn.Write(data)

		if err != nil {
			log.Error("Error sending over UDP conn: ", err)
			continue
		}

		if n != len(data) {
			log.Warn("Not all data written to peer")
		}
	}
}
