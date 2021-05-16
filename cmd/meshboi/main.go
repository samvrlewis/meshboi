package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/samvrlewis/meshboi"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

func main() {

	rollodexCommand := flag.NewFlagSet("rollodex", flag.ExitOnError) // "address book?" "rollodex?"
	ip := rollodexCommand.String("server-ip", "127.0.0.1", "The IP address of the meshboi server")
	port := rollodexCommand.Int("server-port", 12345, "The port of the server")

	clientCommand := flag.NewFlagSet("client", flag.ExitOnError)
	networkName := clientCommand.String("network", "", "The unique network name that identifies the mesh (should be the same on all members in the mesh)")
	tunName := clientCommand.String("tun-name", "tun", "The name to assign to the tun adapter")
	vpnIPPrefixString := clientCommand.String("vpn-ip", "192.168.50.2/24", "The IP address (with subnet) to assign to the tunnel")
	rollodexIPString := clientCommand.String("rollodex-ip", "127.0.0.1", "The IP address of the meshboi server")
	rollodexPort := clientCommand.Int("rollodex-port", 12345, "The port of the server")
	psk := clientCommand.String("psk", "", "The pre shared key to use (should be the same on all members in the mesh)")

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
		if *psk == "" {
			log.Error("psk argument not set. Please set with a secure password")
			clientCommand.PrintDefaults()
			os.Exit(1)
		}

		if *networkName == "" {
			log.Error("network argument not set.")
			clientCommand.PrintDefaults()
			os.Exit(1)
		}

		vpnIPPrefix, err := netaddr.ParseIPPrefix(*vpnIPPrefixString)

		if err != nil {
			log.Fatalln("Error parsing vpn-ip ", err)
		}

		rollodexIP, err := netaddr.ParseIP(*rollodexIPString)

		if err != nil {
			log.Fatalln("Error parsing rollodex-ip ", err)
		}

		mc, err := meshboi.NewMeshBoiClient(*tunName, vpnIPPrefix, rollodexIP, *rollodexPort, *networkName, []byte(*psk))

		if err != nil {
			log.Fatalln("Error starting mesh client ", err)
		}

		mc.Run()
	} else if rollodexCommand.Parsed() {
		addr := &net.UDPAddr{IP: net.ParseIP(*ip), Port: *port}
		conn, err := net.ListenUDP("udp", addr)

		if err != nil {
			log.Fatalln("Error starting listener ", err)
		}

		rollo, err := meshboi.NewRollodex(conn, 5*time.Second, 30*time.Second)

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
