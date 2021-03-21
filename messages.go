package meshboi

import "inet.af/netaddr"

type HeartbeatMessage struct {
	networkName string
}

type NetworkMembers struct {
	addresses []netaddr.IPPort
}

type Version struct {
	version string
}
