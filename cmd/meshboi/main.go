package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/samvrlewis/meshboi"
	"github.com/samvrlewis/meshboi/tun"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

func main() {

	rollodexCommand := flag.NewFlagSet("rollodex", flag.ExitOnError) // "address book?" "rollodex?"
	ip := rollodexCommand.String("server-ip", "127.0.0.1", "The IP address of the meshboi server")
	port := rollodexCommand.Int("server-port", 12345, "The port of the server")

	clientCommand := flag.NewFlagSet("client", flag.ExitOnError)
	tunName := clientCommand.String("tun-name", "tun", "The name to assign to the tunnel")
	tunIP := clientCommand.String("tun-ip", "192.168.50.2/24", "The IP address to assign to the tunnel")
	serverIP := clientCommand.String("server-ip", "127.0.0.1", "The IP address of the meshboi server")
	serverPort := clientCommand.Int("server-port", 12345, "The port of the server")

	if len(os.Args) < 2 {
		log.Fatalln("'server' or 'client' subcommand is required")
	}

	switch os.Args[1] {
	case "rollodex":
		rollodexCommand.Parse(os.Args[2:])
	case "client":
		clientCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if clientCommand.Parsed() {
		tun, err := tun.NewTun(*tunName)

		if err != nil {
			log.Fatalln("Error creating tun: ", err)
		}

		if err := tun.SetLinkUp(); err != nil {
			log.Fatalln("Error setting TUN link up: ", err)
		}

		if err := tun.SetNetwork(*tunIP); err != nil {
			log.Fatalln("Error setting network: ", err)
		}

		if err := tun.SetMtu(1500); err != nil {
			log.Fatalln("Error setting network: ", err)
		}

		listenAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0")}

		multiplexConn, err := meshboi.NewMultiplexedDTLSConn(listenAddr)

		if err != nil {
			log.Fatalln("Error creating multiplexed conn ", err)
		}

		rollodexAddr := &net.UDPAddr{IP: net.ParseIP(*serverIP), Port: *serverPort}
		rollodexConn, err := multiplexConn.GetDialer().Dial(rollodexAddr)

		if err != nil {
			log.Fatalln("Error connecting to rollodex server")
		}

		peerStore := meshboi.NewPeerConnStore()

		peerConnector := meshboi.NewPeerConnector(netaddr.MustParseIPPrefix(*tunIP).IP, multiplexConn.GetListener(), multiplexConn.GetDialer(), peerStore, tun)
		rollodexClient := meshboi.NewRollodexClient("samsNetwork", rollodexConn, time.Duration(5*time.Second), peerConnector.OnNetworkMapUpdate)
		tunRouter := meshboi.NewTunRouter(tun, peerStore)

		go tunRouter.Run()
		go peerConnector.ListenForPeers()
		go rollodexClient.Run()
		defer rollodexClient.Stop()

	} else if rollodexCommand.Parsed() {
		addr := &net.UDPAddr{IP: net.ParseIP(*ip), Port: *port}
		conn, err := net.ListenUDP("udp", addr)

		if err != nil {
			log.Fatalln("Error starting listener ", err)
		}

		rollo, err := meshboi.NewRollodex(conn)

		if err != nil {
			log.Fatalln("Error creating rollodex ", err)
		}
		rollo.Run()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		log.Info("Shutting down")
	}
}
