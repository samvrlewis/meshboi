package meshboi

import (
	"encoding/json"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

type RollodexCallback func(member NetworkMap)

type RollodexClient struct {
	networkName string
	conn        net.Conn
	sendRate    time.Duration
	callback    RollodexCallback
	quit        chan bool
}

func NewRollodexClient(networkName string, conn net.Conn, sendRate time.Duration, callback RollodexCallback) RollodexClient {
	client := RollodexClient{
		conn:     conn,
		sendRate: sendRate,
		callback: callback,
		quit:     make(chan bool),
	}

	return client
}

func (c *RollodexClient) RolloReadLoop() {
	buf := make([]byte, 65535)

	for {
		select {
		case <-c.quit:
			return
		default:
			break
		}

		n, err := c.conn.Read(buf)

		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			log.Warn("Temporary error reading from rolloConn: ", nerr)
			continue
		}

		var members NetworkMap

		if err := json.Unmarshal(buf[:n], &members); err != nil {
			log.Error("Error unmarshalling incoming message: ", err.Error())
			continue
		}

		c.callback(members)
	}
}

func (c *RollodexClient) RolloSendLoop() {
	ticker := time.NewTicker(c.sendRate)
	for {
		select {
		case <-c.quit:
			return
		case <-ticker.C:
			break
		}

		heartbeat := HeartbeatMessage{NetworkName: c.networkName}
		b, err := json.Marshal(heartbeat)
		if err != nil {
			log.Fatalln("Error marshalling JSON heartbeat message: ", err)
		}

		_, err = c.conn.Write(b)

		if err != nil {
			log.Error("Error sending heartbeat over the rollo conn: ", err)
		}
	}
}

func (c *RollodexClient) Stop() {
	c.quit <- true
}
