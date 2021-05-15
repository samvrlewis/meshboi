package meshboi

import (
	"net"
	"time"

	"github.com/samvrlewis/meshboi/tun"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

type MeshboiClient struct {
	peerStore     *PeerConnStore
	rolloClient   RollodexClient
	tunRouter     TunRouter
	peerConnector PeerConnector
}

func NewMeshBoiClient(tunName string, tunIP string, serverIP net.IP, serverPort int, psk []byte) (*MeshboiClient, error) {
	tun, err := tun.NewTun(tunName)

	if err != nil {
		log.Error("Error creating tun: ", err)
		return nil, err
	}

	if err := tun.SetLinkUp(); err != nil {
		log.Error("Error setting TUN link up: ", err)
		return nil, err
	}

	if err := tun.SetNetwork(tunIP); err != nil {
		log.Error("Error setting network: ", err)
		return nil, err
	}

	if err := tun.SetMtu(1500); err != nil {
		log.Error("Error setting network: ", err)
		return nil, err
	}

	vpnIpPrefix, err := netaddr.ParseIPPrefix(tunIP)
	if err != nil {
		log.Error("Error parsing vpn IP ", err)
		return nil, err
	}

	listenAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0")}
	dtlsConfig := getDtlsConfig(vpnIpPrefix.IP, psk)

	multiplexConn, err := NewMultiplexedDTLSConn(listenAddr, dtlsConfig)

	if err != nil {
		log.Error("Error creating multiplexed conn ", err)
		return nil, err
	}

	rollodexAddr := &net.UDPAddr{IP: serverIP, Port: serverPort}
	rollodexConn, err := multiplexConn.Dial(rollodexAddr)

	if err != nil {
		log.Error("Error connecting to rollodex server")
		return nil, err
	}

	mc := MeshboiClient{}

	mc.peerStore = NewPeerConnStore()
	mc.peerConnector = NewPeerConnector(multiplexConn, mc.peerStore, tun)
	mc.rolloClient = NewRollodexClient("samsNetwork", rollodexConn, time.Duration(5*time.Second), mc.peerConnector.OnNetworkMapUpdate)
	mc.tunRouter = NewTunRouter(tun, mc.peerStore)

	return &mc, nil
}

func (mc *MeshboiClient) Run() {
	go mc.tunRouter.Run()
	go mc.peerConnector.ListenForPeers()
	go mc.rolloClient.Run()
	defer mc.rolloClient.Stop()

	for {

	}
}
