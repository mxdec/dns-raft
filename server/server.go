package server

import (
	"log"
	"net"
	"os"

	"github.com/maxdcr/pagesjaunes/store"
)

// TCPServer contains TCP Listener
type TCPServer struct {
	// TCP Listener
	lst net.Listener

	// Pointer to KV Store
	kvs *store.Store

	logger *log.Logger
}

// InitTCP initialize a TCP server
func InitTCP(kvs *store.Store, tcpAddr string) {
	var tcp TCPServer
	var err error

	tcp.logger = log.New(os.Stderr, "", log.LstdFlags)
	tcp.logger.Printf("setting TCP listener to %s", tcpAddr)
	tcp.kvs = kvs
	tcp.lst, err = net.Listen("tcp", tcpAddr)
	if err != nil {
		tcp.logger.Printf("Failed to set TCP listener %s", err.Error())
	}
	// accept connections
	go func() {
		for {
			// accept new client connect and perform
			conn, err := tcp.lst.Accept()
			if err != nil {
				tcp.logger.Printf("could't accept client, err: %v", err)
				continue
			}
			// handle conn
			TCPHandler(conn, tcp.kvs)
		}
	}()
}
