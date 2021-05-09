package meshboi

import (
	"sync"

	"inet.af/netaddr"
)

type PeerConnStore struct {
	peersByOutsideIPPort map[netaddr.IPPort]*PeerConn
	peersByInsideIP      map[netaddr.IP]*PeerConn
	lock                 sync.RWMutex
}

func NewPeerConnStore() *PeerConnStore {
	s := &PeerConnStore{}
	s.peersByInsideIP = make(map[netaddr.IP]*PeerConn)
	s.peersByOutsideIPPort = make(map[netaddr.IPPort]*PeerConn)

	return s
}

func (p *PeerConnStore) Add(peer *PeerConn) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.peersByInsideIP[peer.insideIP] = peer
	p.peersByOutsideIPPort[peer.outsideAddr] = peer
}

func (p *PeerConnStore) GetByInsideIp(insideIP netaddr.IP) (*PeerConn, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peer, ok := p.peersByInsideIP[insideIP]

	return peer, ok
}

func (p *PeerConnStore) GetByOutsideIpPort(outsideIPPort netaddr.IPPort) (*PeerConn, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peer, ok := p.peersByOutsideIPPort[outsideIPPort]

	return peer, ok
}

func (p *PeerConnStore) RemoveByOutsideIPPort(outsideIPPort netaddr.IPPort) bool {
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
