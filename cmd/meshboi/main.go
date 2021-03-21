package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/samvrlewis/meshboi"
	"github.com/samvrlewis/meshboi/tun"
	"inet.af/netaddr"
)

const (
	MTU_SIZE = 1500
)

type registerMessage struct {
	vpnIp string
}

type clientsMessage struct {
	// mapping of internal IPs to external IP and ports
}

type allIps struct {
	Members []netaddr.IPPort
}

func main() {

	serverCommand := flag.NewFlagSet("server", flag.ExitOnError) // "address book?" "rollodex?"
	ip := serverCommand.String("server-ip", "127.0.0.1", "The IP address of the meshboi server")
	port := serverCommand.Int("server-port", 12345, "The port of the server")

	clientCommand := flag.NewFlagSet("client", flag.ExitOnError)
	tunName := clientCommand.String("tun-name", "tun", "The name to assign to the tunnel")
	tunIP := clientCommand.String("tun-ip", "192.168.50.2/24", "The IP address to assign to the tunnel")
	serverIP := clientCommand.String("server-ip", "localhost", "The IP address of the meshboi server")
	serverPort := clientCommand.Int("server-port", 12345, "The port of the server")

	if len(os.Args) < 2 {
		fmt.Println("server or client subcommand is required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		serverCommand.Parse(os.Args[2:])
	case "client":
		clientCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if clientCommand.Parsed() {
		// log.Println(*serverIP)
		// log.Println(*serverPort)

		tun, err := tun.NewTun(*tunName)

		if err != nil {
			log.Fatalln("err ", err)
		}

		if err := tun.SetLinkUp(); err != nil {
			log.Fatalln("Error setting TUN link up: ", err)
		}

		if err := tun.SetNetwork(*tunIP); err != nil {
			log.Fatalln("Error setting network: ", err)
		}

		if err := tun.SetMtu(MTU_SIZE); err != nil {
			log.Fatalln("Error setting network: ", err)
		}

		client := meshboi.NewClient(*serverIP, *serverPort, tun)

		go client.RolloReadLoop()
		go client.RolloSendLoop()

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		select {
		case <-c:
			log.Println("Shutting down")
		}
	} else if serverCommand.Parsed() {
		log.Println("server")
		addr := &net.UDPAddr{IP: net.ParseIP(*ip), Port: *port}
		log.Println(addr.IP)
		conn, err := net.ListenUDP("udp", addr)

		if err != nil {
			log.Fatalln("err ", err)
		}

		rollo, err := meshboi.NewRollodex(conn)

		if err != nil {
			log.Fatalln("err ", err)
		}
		log.Println("server")
		rollo.Run()
	}
}
