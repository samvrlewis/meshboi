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
	rolloClient   RolodexClient
	tunRouter     TunRouter
	peerConnector PeerConnector
}

func NewMeshBoiClient(tun TunConn, vpnIpPrefix netaddr.IPPrefix, rolodexIP netaddr.IP, rolodexPort int, networkName string, meshPSK []byte) (*MeshboiClient, error) {
	listenAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0")}
	dtlsConfig := getDtlsConfig(vpnIpPrefix.IP, meshPSK)

	multiplexConn, err := NewMultiplexedDTLSConn(listenAddr, dtlsConfig)

	if err != nil {
		log.Error("Error creating multiplexed conn ", err)
		return nil, err
	}

	rolodexAddr := &net.UDPAddr{IP: rolodexIP.IPAddr().IP, Port: rolodexPort}
	rolodexConn, err := multiplexConn.Dial(rolodexAddr)

	if err != nil {
		log.Error("Error connecting to rolodex server")
		return nil, err
	}

	mc := MeshboiClient{}

	mc.peerStore = NewPeerConnStore()
	mc.peerConnector = NewPeerConnector(multiplexConn, mc.peerStore, tun)
	mc.rolloClient = NewRolodexClient(networkName, rolodexConn, time.Duration(5*time.Second), mc.peerConnector.OnNetworkMapUpdate)
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
