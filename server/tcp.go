package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/maxdcr/pagesjaunes/store"
)

// TCPConn contains incoming connexion and command with args
type TCPConn struct {
	conn   net.Conn
	store  *store.Store
	logger *log.Logger
	verb   string
	args   []string
}

// TCPHandler triggers handler in a go-routine
func TCPHandler(conn net.Conn, store *store.Store) {
	c := &TCPConn{
		conn:   conn,
		store:  store,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
	go c.handleRequest()
}

// Handles incoming requests.
func (c *TCPConn) handleRequest() {
	input, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}
	tmp := strings.TrimSpace(string(input))
	c.logger.Printf("read |%s| from input", tmp)

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
func (c *TCPConn) handleCmd(cmd []string) string {
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

func (c *TCPConn) handleJoin() string {
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

func (c *TCPConn) handleLeave() string {
	if len(c.args) != 1 {
		return "ERROR"
	}

	nodeID := c.args[0]
	if err := c.store.Leave(nodeID); err != nil {
		return err.Error()
	}
	return "SUCCESS"
}

func (c *TCPConn) handleGet() string {
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

func (c *TCPConn) handleSet() string {
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

func (c *TCPConn) handleDel() string {
	if len(c.args) != 1 {
		return "ERROR"
	}

	k := c.args[0]
	if err := c.store.Delete(k); err != nil {
		return err.Error()
	}
	return "SUCCESS"
}
