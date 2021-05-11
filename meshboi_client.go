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

func NewMeshBoiClient(tunName string, tunIP string, serverIP net.IP, serverPort int) *MeshboiClient {
	tun, err := tun.NewTun(tunName)

	if err != nil {
		log.Fatalln("Error creating tun: ", err)
	}

	if err := tun.SetLinkUp(); err != nil {
		log.Fatalln("Error setting TUN link up: ", err)
	}

	if err := tun.SetNetwork(tunIP); err != nil {
		log.Fatalln("Error setting network: ", err)
	}

	if err := tun.SetMtu(1500); err != nil {
		log.Fatalln("Error setting network: ", err)
	}

	listenAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0")}

	multiplexConn, err := NewMultiplexedDTLSConn(listenAddr)

	if err != nil {
		log.Fatalln("Error creating multiplexed conn ", err)
	}

	rollodexAddr := &net.UDPAddr{IP: serverIP, Port: serverPort}
	rollodexConn, err := multiplexConn.GetDialer().Dial(rollodexAddr)

	if err != nil {
		log.Fatalln("Error connecting to rollodex server")
	}

	mc := MeshboiClient{}

	mc.peerStore = NewPeerConnStore()
	mc.peerConnector = NewPeerConnector(netaddr.MustParseIPPrefix(tunIP).IP, multiplexConn.GetListener(), multiplexConn.GetDialer(), mc.peerStore, tun)
	mc.rolloClient = NewRollodexClient("samsNetwork", rollodexConn, time.Duration(5*time.Second), mc.peerConnector.OnNetworkMapUpdate)
	mc.tunRouter = NewTunRouter(tun, mc.peerStore)

	return &mc
}

func (mc *MeshboiClient) Run() {
	go mc.tunRouter.Run()
	go mc.peerConnector.ListenForPeers()
	go mc.rolloClient.Run()
	defer mc.rolloClient.Stop()

	for {

	}
}
