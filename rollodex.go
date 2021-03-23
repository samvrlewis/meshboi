package meshboi

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	"inet.af/netaddr"
)

type rollodex struct {
	conn     *net.UDPConn
	networks map[string]*meshNetwork
}

const TimeOutSecs = 30

type meshNetwork struct {
	// make of IP address to last seen time
	members     map[netaddr.IPPort]time.Time
	membersLock sync.RWMutex
	rollo       *rollodex
}

func (m *meshNetwork) register(addr netaddr.IPPort) {
	println("Registering ", ":", addr.Port)
	m.members[addr] = time.Now()
}

func (r *rollodex) getNetwork(networkName string) *meshNetwork {
	if network, ok := r.networks[networkName]; ok {
		return network
	}

	network := &meshNetwork{}
	network.members = make(map[netaddr.IPPort]time.Time)
	network.rollo = r
	r.networks[networkName] = network

	go network.Serve()

	return network
}

func NewRollodex(conn *net.UDPConn) (*rollodex, error) {

	// laddr, err := net.ResolveUDPAddr("udp", address)
	// if err != nil {
	// 	return nil, err
	// }

	println("hello")
	rollo := &rollodex{}
	rollo.conn = conn
	rollo.networks = make(map[string]*meshNetwork)

	return rollo, nil
}

func (r *rollodex) Run() {
	println("waiting for data")
	buf := make([]byte, 65535)
	for {
		println("waiting for data")
		n, addr, err := r.conn.ReadFromUDP(buf)

		println("Got some data")

		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			println("temp error")
			continue
		}

		println("Got some data")
		println(string(buf[:n]))

		var message HeartbeatMessage

		if err := json.Unmarshal(buf[:n], &message); err != nil {
			println("Error unmarshalling ", err.Error())
			continue
		}

		mesh := r.getNetwork(message.NetworkName)
		ipPort, ok := netaddr.FromStdAddr(addr.IP, addr.Port, "")

		if !ok {
			continue
		}

		mesh.register(ipPort)
	}
}

func (mesh *meshNetwork) timeOutInactiveMembers() {
	mesh.membersLock.Lock()
	defer mesh.membersLock.Unlock()

	now := time.Now()

	// todo: Can't iterate and delete from a map at the same time
	for member := range mesh.members {
		timeSinceLastActive := now.Sub(mesh.members[member])

		if timeSinceLastActive.Seconds() > TimeOutSecs {
			println("Removing due to timeout ", member.Port)
			delete(mesh.members, member)
		}
	}
}

// Serve sends out messages to each member so that they're aware of other members they can connect to
// It also serves as a heart beat of sorts from the rollodex to the member
func (mesh *meshNetwork) Serve() {
	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan int)
	for {
		select {
		case <-ticker.C:
			break
		case <-quit:
			ticker.Stop()
			return
		}

		mesh.timeOutInactiveMembers()

		mesh.membersLock.RLock()
		memberIps := make([]netaddr.IPPort, 0, len(mesh.members))
		for member := range mesh.members {
			memberIps = append(memberIps, member)
		}

		memberMessage := NetworkMap{Addresses: memberIps}
		memberMessage.YourIndex = 0

		// todo: This assumes that mesh.members will be in the same order as it
		// was when it was serialised. Which it probably won't be. Could
		// probably iterate over memberIPs more safely instead.
		for k := range mesh.members {
			b, err := json.Marshal(memberMessage)
			if err != nil {
				panic(err)
			}
			mesh.rollo.conn.WriteToUDP(b, k.UDPAddr())
			memberMessage.YourIndex += 1
		}
		mesh.membersLock.RUnlock()
	}
}
