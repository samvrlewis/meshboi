package meshboi

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"inet.af/netaddr"
)

type rollodex struct {
	conn            *net.UDPConn
	networks        map[string]*meshNetwork
	sendInterval    time.Duration
	timeOutDuration time.Duration
}

const TimeOutSecs = 30

type meshNetwork struct {
	// make of IP address to last seen time
	members     map[netaddr.IPPort]time.Time
	membersLock sync.RWMutex
	rollo       *rollodex
	newMember   chan struct{}
}

func (m *meshNetwork) register(addr netaddr.IPPort) {
	m.membersLock.Lock()
	_, ok := m.members[addr]
	m.members[addr] = time.Now()
	m.membersLock.Unlock()

	if !ok {
		log.WithFields(log.Fields{
			"address": addr,
		}).Info("Registering new mesh member")
		m.newMember <- struct{}{}
	}
}

func (r *rollodex) getNetwork(networkName string) *meshNetwork {
	if network, ok := r.networks[networkName]; ok {
		return network
	}

	network := &meshNetwork{}
	network.members = make(map[netaddr.IPPort]time.Time)
	network.rollo = r
	network.newMember = make(chan struct{})
	r.networks[networkName] = network

	go network.Serve()

	return network
}

func NewRollodex(conn *net.UDPConn, sendInterval time.Duration, timeOutDuration time.Duration) (*rollodex, error) {
	rollo := &rollodex{}
	rollo.conn = conn
	rollo.sendInterval = sendInterval
	rollo.timeOutDuration = timeOutDuration
	rollo.networks = make(map[string]*meshNetwork)

	return rollo, nil
}

func (r *rollodex) Run() {
	buf := make([]byte, 65535)
	for {
		n, addr, err := r.conn.ReadFromUDP(buf)

		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			log.Warn("Temporary error reading data: ", nerr)
			continue
		}

		var message HeartbeatMessage

		if err := json.Unmarshal(buf[:n], &message); err != nil {
			log.Error("Error unmarshalling ", err)
			continue
		}

		mesh := r.getNetwork(message.NetworkName)
		ipPort, ok := netaddr.FromStdAddr(addr.IP, addr.Port, "")

		if !ok {
			log.Error("Error converting to netaddr ", err)
			continue
		}

		mesh.register(ipPort)
	}
}

func (mesh *meshNetwork) timeOutInactiveMembers() {
	mesh.membersLock.Lock()
	defer mesh.membersLock.Unlock()

	now := time.Now()

	for member := range mesh.members {
		timeSinceLastActive := now.Sub(mesh.members[member])

		if timeSinceLastActive > mesh.rollo.timeOutDuration {
			log.WithFields(log.Fields{
				"address": member.IP,
			}).Info("Removing member due to timeout")
			delete(mesh.members, member)
		}
	}
}

// Serve sends out messages to each member so that they're aware of other members they can connect to
// It also serves as a heart beat of sorts from the rollodex to the member
func (mesh *meshNetwork) Serve() {
	ticker := time.NewTicker(mesh.rollo.sendInterval)
	quit := make(chan int)
	for {
		// Send out an update both periodically, and on the event of a new member joining
		select {
		case <-ticker.C:
			break
		case <-mesh.newMember:
			break
		case <-quit:
			ticker.Stop()
			return
		}

		// reset the ticker in case we're sending an update due to a new member joining
		ticker.Reset(mesh.rollo.sendInterval)

		mesh.timeOutInactiveMembers()

		mesh.membersLock.RLock()
		memberIps := make([]netaddr.IPPort, 0, len(mesh.members))
		for member := range mesh.members {
			memberIps = append(memberIps, member)
		}

		memberMessage := NetworkMap{Addresses: memberIps}
		memberMessage.YourIndex = 0

		for _, member := range memberIps {
			b, err := json.Marshal(memberMessage)
			if err != nil {
				panic(err)
			}
			mesh.rollo.conn.WriteToUDP(b, member.UDPAddr())
			memberMessage.YourIndex += 1
		}
		mesh.membersLock.RUnlock()
	}
}
