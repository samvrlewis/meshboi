package meshboi

import (
	"net"
	"sync"
	"time"

	"github.com/samvrlewis/meshboi/tun"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

// Represents a peer that has been connected to
type Peer struct {
	// The IP address within the VPN
	insideIP netaddr.IP

	// The IP address over the internet
	outsideAddr netaddr.IPPort

	// Time of last contact
	lastContacted time.Time
	conn          net.Conn
	outgoing      chan []byte
	tun           *tun.Tun
}

func NewPeer(insideIP netaddr.IP, outsideAddr netaddr.IPPort, conn net.Conn, tun *tun.Tun) Peer {
	return Peer{
		insideIP:      insideIP,
		outsideAddr:   outsideAddr,
		conn:          conn,
		tun:           tun,
		lastContacted: time.Now(),
		outgoing:      make(chan []byte),
	}
}

func (p *Peer) QueueData(data []byte) {
	p.outgoing <- data
}

func (p *Peer) readLoop() {
	b := make([]byte, bufSize)
	for {
		n, err := p.conn.Read(b)
		if err != nil {
			panic(err)
		}
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
func (p *Peer) sendLoop() {
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

type PeerStore struct {
	peersByOutsideIPPort map[netaddr.IPPort]*Peer
	peersByInsideIP      map[netaddr.IP]*Peer
	lock                 sync.RWMutex
}

func NewPeerStore() *PeerStore {
	s := &PeerStore{}
	s.peersByInsideIP = make(map[netaddr.IP]*Peer)
	s.peersByOutsideIPPort = make(map[netaddr.IPPort]*Peer)

	return s
}

func (p *PeerStore) Add(peer *Peer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.peersByInsideIP[peer.insideIP] = peer
	p.peersByOutsideIPPort[peer.outsideAddr] = peer
}

func (p *PeerStore) GetByInsideIp(insideIP netaddr.IP) (*Peer, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peer, ok := p.peersByInsideIP[insideIP]

	return peer, ok
}

func (p *PeerStore) GetByOutsideIpPort(outsideIPPort netaddr.IPPort) (*Peer, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peer, ok := p.peersByOutsideIPPort[outsideIPPort]

	return peer, ok
}

func (p *PeerStore) RemoveByOutsideIPPort(outsideIPPort netaddr.IPPort) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	peer, ok := p.GetByOutsideIpPort(outsideIPPort)

	if !ok {
		return false
	}

	insideIp := peer.insideIP

	delete(p.peersByInsideIP, insideIp)
	delete(p.peersByOutsideIPPort, outsideIPPort)

	return true
}
