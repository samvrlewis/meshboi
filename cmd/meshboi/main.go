package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/samvrlewis/meshboi"
	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

const defaultPort = 6264 // "mboi" on a telelphone dialpad :)

const usage = `usage: meshboi <cmd> args

Command can be one of

	rolodex	Start meshboi in rollodex mode
	client		Join as client in an existing peer to peer mesh

More information on both commands and the arguments needed can be found with
meshboi <cmd> -help. (eg: meshboi rolodex -help).`

func printUsage() {
	fmt.Println(usage)
	os.Exit(1)
}

func main() {

	rolodexCommand := flag.NewFlagSet("rolodex", flag.ExitOnError)
	ip := rolodexCommand.String("listen-address", "0.0.0.0", "The IP address for the rolodex to listen on")
	port := rolodexCommand.Int("listen-port", defaultPort, "The port of for the rolodex to listen on")

	clientCommand := flag.NewFlagSet("client", flag.ExitOnError)
	networkName := clientCommand.String("network", "", "The unique network name that identifies the mesh (should be the same on all members in the mesh)")
	tunName := clientCommand.String("tun-name", "tun", "The name to assign to the tun adapter")
	tunMtu := clientCommand.Int("tun-mtu", 1200, "The MTU of the tun")
	vpnIPPrefixString := clientCommand.String("vpn-ip", "", "The IP address (with subnet) to assign to the tunnel eg: 192.168.50.1/24")
	rolodexAddr := clientCommand.String("rolodex-address", "rolodex.samlewis.me", "The IP address of the meshboi server")
	rolodexPort := clientCommand.Int("rolodex-port", defaultPort, "The port of the server")
	psk := clientCommand.String("psk", "", "The pre shared key to use (should be the same on all members in the mesh)")

	if len(os.Args) < 2 {
		printUsage()
	}

	switch os.Args[1] {
	case "rolodex":
		rolodexCommand.Parse(os.Args[2:])
	case "client":
		clientCommand.Parse(os.Args[2:])
	default:
		printUsage()
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

		if *vpnIPPrefixString == "" {
			log.Error("vpn-ip argument not set.")
			clientCommand.PrintDefaults()
			os.Exit(1)
		}

		vpnIPPrefix, err := netaddr.ParseIPPrefix(*vpnIPPrefixString)

		tun, err := meshboi.NewTunWithConfig(*tunName, vpnIPPrefix.String(), *tunMtu)

		if err != nil {
			log.Fatalln("Error creating tun: ", err)
		}

		rolodexStdIP, err := net.ResolveIPAddr("ip", *rolodexAddr)

		if err != nil {
			log.Fatalln("Error parsing rolodex-address ", err)
		}

		rolodexIP, ok := netaddr.FromStdIP(rolodexStdIP.IP)

		if !ok {
			log.Fatalln("Error converting to netaddr IP")
		}

		mc, err := meshboi.NewMeshBoiClient(tun, vpnIPPrefix, rolodexIP, *rolodexPort, *networkName, []byte(*psk))

		if err != nil {
			log.Fatalln("Error starting mesh client ", err)
		}

		mc.Run()
	} else if rolodexCommand.Parsed() {
		addr := &net.UDPAddr{IP: net.ParseIP(*ip), Port: *port}
		conn, err := net.ListenUDP("udp", addr)

		if err != nil {
			log.Fatalln("Error starting listener ", err)
		}

		rollo, err := meshboi.NewRolodex(conn, 5*time.Second, 30*time.Second)

		if err != nil {
			log.Fatalln("Error creating rolodex ", err)
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
