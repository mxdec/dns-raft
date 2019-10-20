package tcp

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"

	"github.com/maxdrc/dns-raft/store"
)

// Req contains incoming connexion and command with args
type Req struct {
	conn   net.Conn
	store  *store.Store
	logger *log.Logger
	verb   string
	args   []string
}

// Handler triggers handler in a go-routine
func Handler(conn net.Conn, store *store.Store) {
	c := &Req{
		conn:   conn,
		store:  store,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
	go c.handleRequest()
}

// Handles incoming requests.
func (c *Req) handleRequest() {
	input, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		c.logger.Println("error reading:", err.Error())
		return
	}
	tmp := strings.TrimSpace(string(input))
	c.logger.Printf("new tcp msg: %s\n", tmp)

	// trim spaces
	cmd := strings.Fields(tmp)
	// handle command
	rsp := c.handleCmd(cmd)
	// send a response back
	c.conn.Write([]byte(rsp))
	// close the connection
	c.conn.Close()
}

// Select the handler.
func (c *Req) handleCmd(cmd []string) string {
	if len(cmd) == 0 {
		return "ERROR"
	}
	c.verb = strings.ToLower(cmd[0])
	c.args = cmd[1:]

	c.logger.Printf("processing %s command", cmd)
	switch c.verb {
	case "join":
		return c.handleJoin()
	case "leave":
		return c.handleLeave()
	case "get":
		return c.handleGet()
	case "set":
		return c.handleSet()
	case "del":
		return c.handleDel()
	case "ping":
		return "PONG"
	default:
		return "ERROR"
	}
}

func (c *Req) handleJoin() string {
	if len(c.args) != 2 {
		return "ERROR"
	}

	raftAddr := c.args[0]
	nodeID := c.args[1]
	if err := c.store.Join(nodeID, raftAddr); err != nil {
		return err.Error()
	}
	return "SUCCESS"
}

func (c *Req) handleLeave() string {
	if len(c.args) != 1 {
		return "ERROR"
	}

	nodeID := c.args[0]
	if err := c.store.Leave(nodeID); err != nil {
		return err.Error()
	}
	return "SUCCESS"
}

func (c *Req) handleGet() string {
	if len(c.args) != 1 {
		return "ERROR"
	}

	k := c.args[0]
	v, ok := c.store.Get(k)
	if !ok {
		return "ERROR"
	}
	return v
}

func (c *Req) handleSet() string {
	if len(c.args) != 2 {
		return "ERROR"
	}

	k := c.args[0]
	v := c.args[1]
	if err := c.store.Set(k, v); err != nil {
		return err.Error()
	}
	return "SUCCESS"
}

func (c *Req) handleDel() string {
	if len(c.args) != 1 {
		return "ERROR"
	}

	k := c.args[0]
	if err := c.store.Delete(k); err != nil {
		return err.Error()
	}
	return "SUCCESS"
}
