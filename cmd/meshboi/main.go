package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/samvrlewis/meshboi/tun"
)

const (
	MTU_SIZE = 150
)

func main() {
	tunName := flag.String("tun-name", "tun", "The name to assign to the tunnel")
	tunIP := flag.String("tun-ip", "192.168.50.2/24", "The IP address to assign to the tunnel")

	flag.Parse()
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

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		log.Println("Shutting down")
	}
}
