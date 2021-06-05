package meshboi

import (
	"net"

	"golang.org/x/net/ipv4"
	"inet.af/netaddr"

	log "github.com/sirupsen/logrus"
)

type TunRouter struct {
	tun   TunConn
	store *PeerConnStore
	quit  chan struct{}
}

func NewTunRouter(tun TunConn, store *PeerConnStore) TunRouter {
	return TunRouter{
		tun:   tun,
		store: store,
	}
}

func (tr *TunRouter) Run() {
	packet := make([]byte, bufSize)

	for {
		n, err := tr.tun.Read(packet)
		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			log.Warn("Temporary error reading from tun device, continuing: ", nerr)
			continue
		}

		if err != nil {
			log.Fatalln("Serious error reading from tun device: ", err)
			break
		}

		header, err := ipv4.ParseHeader(packet[:n])

		if err != nil {
			log.Error("Error parsing ipv4 header of tun packet: ", err)
			continue
		}

		vpnIP, ok := netaddr.FromStdIP(header.Dst)

		if !ok {
			log.Error("Error converting to netaddr IP")
			continue
		}

		peer, ok := tr.store.GetByInsideIp(vpnIP)

		if !ok {
			log.Warn("Dropping data destined for ", vpnIP)
			continue
		}

		msg := make([]byte, n)
		copy(msg, packet[:n])

		peer.QueueData(msg)
	}
}

func (tr *TunRouter) Stop() error {
	return tr.tun.Close()
}
