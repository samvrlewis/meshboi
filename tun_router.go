package meshboi

import (
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
		if err != nil {
			log.Error("Error reading from tun device: ", err)
			continue
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
			log.Info("Dropping data destined for ", vpnIP)
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
