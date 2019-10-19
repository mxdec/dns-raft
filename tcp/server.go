package tcp

import (
	"log"
	"net"
	"os"

	"github.com/maxdcr/dns-raft/store"
)

// TCP wrapper
type TCP struct {
	addr   string
	lst    net.Listener
	kvs    *store.Store
	logger *log.Logger
}

// NewTCP initialize a TCP server
func NewTCP(kvs *store.Store, tcpAddr string) *TCP {
	return &TCP{
		kvs:    kvs,
		addr:   tcpAddr,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

// Start initialize a TCP server
func (t *TCP) Start() {
	var err error

	t.logger.Printf("setting TCP listener to %s", t.addr)
	t.lst, err = net.Listen("tcp", t.addr)
	if err != nil {
		t.logger.Fatalf("Failed to set TCP listener %v", err)
	}
	// accept connections
	go func() {
		for {
			// accept new client connect and perform
			conn, err := t.lst.Accept()
			if err != nil {
				t.logger.Printf("could't accept client, err: %v", err)
				continue
			}
			r := &Req{
				conn:   conn,
				store:  t.kvs,
				logger: log.New(os.Stderr, "", log.LstdFlags),
			}
			go r.handleRequest()
		}
	}()
}
