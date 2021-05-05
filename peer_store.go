package meshboi

import (
	"sync"

	"inet.af/netaddr"
)

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
