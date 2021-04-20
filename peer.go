package meshboi

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

type Peer struct {
	tunIP         netaddr.IP
	remoteIP      netaddr.IPPort
	lastContacted time.Time
	conn          net.Conn
	outgoing      chan []byte
	member        *MeshMember
}

func (p *Peer) readLoop() {
	b := make([]byte, bufSize)
	for {
		n, err := p.conn.Read(b)
		if err != nil {
			panic(err)
		}
		log.Info("Got message: ", b[:n])

		written, err := p.member.tun.Write(b[:n])

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

func (p *Peer) addData(data []byte) {
	p.outgoing <- data
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

	p.peersByInsideIP[peer.tunIP] = peer
	p.peersByOutsideIPPort[peer.remoteIP] = peer
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

	insideIp := peer.tunIP

	delete(p.peersByInsideIP, insideIp)
	delete(p.peersByOutsideIPPort, outsideIPPort)

	return true
}
