package meshboi

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

type MeshboiClient struct {
	peerStore     *PeerConnStore
	rolloClient   RollodexClient
	tunRouter     TunRouter
	peerConnector PeerConnector
}

func NewMeshBoiClient(tun TunConn, vpnIpPrefix netaddr.IPPrefix, rollodexIP netaddr.IP, rollodexPort int, networkName string, meshPSK []byte) (*MeshboiClient, error) {
	listenAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0")}
	dtlsConfig := getDtlsConfig(vpnIpPrefix.IP, meshPSK)

	multiplexConn, err := NewMultiplexedDTLSConn(listenAddr, dtlsConfig)

	if err != nil {
		log.Error("Error creating multiplexed conn ", err)
		return nil, err
	}

	rollodexAddr := &net.UDPAddr{IP: rollodexIP.IPAddr().IP, Port: rollodexPort}
	rollodexConn, err := multiplexConn.Dial(rollodexAddr)

	if err != nil {
		log.Error("Error connecting to rollodex server")
		return nil, err
	}

	mc := MeshboiClient{}

	mc.peerStore = NewPeerConnStore()
	mc.peerConnector = NewPeerConnector(multiplexConn, mc.peerStore, tun)
	mc.rolloClient = NewRollodexClient(networkName, rollodexConn, time.Duration(5*time.Second), mc.peerConnector.OnNetworkMapUpdate)
	mc.tunRouter = NewTunRouter(tun, mc.peerStore)

	return &mc, nil
}

func (mc *MeshboiClient) Run() {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		mc.tunRouter.Run()
		wg.Done()
	}()

	go func() {
		mc.peerConnector.ListenForPeers()
		wg.Done()
	}()

	go func() {
		mc.rolloClient.Run()
		wg.Done()
	}()

	wg.Wait()
}

func (mc *MeshboiClient) Stop() {
	mc.rolloClient.Stop()
	mc.peerConnector.Stop()
	mc.tunRouter.Stop()

}
