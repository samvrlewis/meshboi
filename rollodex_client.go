package meshboi

import (
	"encoding/json"
	"net"
	"sync"
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
	wg          *sync.WaitGroup
}

func NewRollodexClient(networkName string, conn net.Conn, sendRate time.Duration, callback RollodexCallback) RollodexClient {
	client := RollodexClient{
		networkName: networkName,
		conn:        conn,
		sendRate:    sendRate,
		callback:    callback,
		quit:        make(chan bool),
		wg:          &sync.WaitGroup{},
	}

	return client
}

func (c *RollodexClient) Run() {
	go c.readLoop()
	go c.sendLoop()

	c.wg.Wait()
}

func (c *RollodexClient) readLoop() {
	c.wg.Add(1)
	defer c.wg.Done()

	buf := make([]byte, 65535)
	for {
		n, err := c.conn.Read(buf)

		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			log.Warn("Temporary error reading from rolloConn: ", nerr)
			continue
		}

		if err != nil {
			log.Error("Unrecoverable error: ", err)
			break
		}

		var members NetworkMap

		if err := json.Unmarshal(buf[:n], &members); err != nil {
			log.Error("Error unmarshalling incoming message: ", err.Error())
			continue
		}

		c.callback(members)
	}
}

func (c *RollodexClient) sendLoop() {
	c.wg.Add(1)
	defer c.wg.Done()

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
	c.conn.Close()
}
