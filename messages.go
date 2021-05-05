package meshboi

import "inet.af/netaddr"

type HeartbeatMessage struct {
	NetworkName string
}

type NetworkMap struct {
	Addresses []netaddr.IPPort
	YourIndex int
}
