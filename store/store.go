package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/raft"
)

const (
	// tcp messages
	joinMsg  = "kv join %s %s\n"
	leaveMsg = "kv leave %s\n"
	setMsg   = "kv set %s %s\n"
	delMsg   = "kv del %s\n"

	// tcp response buffer size
	rspBuffSize = 1024
)

// Store is a simple key-value store, where all changes are made via Raft consensus.
type Store struct {
	RaftAddr string
	RaftID   string

	// TCP listener
	ln net.Listener

	// raft
	fsm       *fsm       // The finite-state machine
	raft      *raft.Raft // The consensus mechanism
	raftLayer *raftLayer // The TCP wrapper for Raft

	// logger
	logger *log.Logger
}

// InitStore returns an initialized KV store
func InitStore(addr, join, id string) *Store {
	var store Store
	var err error

	store.RaftAddr = addr
	store.RaftID = id
	store.logger = log.New(os.Stderr, "", log.LstdFlags)

	// Create the FSM.
	store.fsm = newFSM()

	// Initialize TCP
	store.ln, err = net.Listen("tcp", store.RaftAddr)
	if err != nil {
		store.logger.Println(err.Error())
		panic(err.Error())
	}

	// start Raft
	if err := store.initRaft(join); err != nil {
		store.logger.Println(err.Error())
		panic(err.Error())
	}

	// Start listening for requests.
	go store.start(store.ln)

	return &store
}

// Open initializes the store
func (s *Store) initRaft(join string) error {
	// Setup Raft configuration.
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(s.RaftID)

	// set Raft and KV connections on same TCP port
	s.raftLayer = &raftLayer{
		addr:   s.ln.Addr().(*net.TCPAddr),
		connCh: make(chan net.Conn),
	}
	trans := raft.NewNetworkTransport(s.raftLayer, 3, 10*time.Second, os.Stderr)

	// Create the snapshot store, log store and stable store in memory
	snap := raft.NewInmemSnapshotStore()
	log := raft.NewInmemStore()
	stable := raft.NewInmemStore()

	// Instantiate the Raft systems
	r, err := raft.NewRaft(config, s.fsm, log, stable, snap, trans)
	if err != nil {
		return err
	}
	s.raft = r

	if len(join) == 0 {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: trans.LocalAddr(),
				},
			},
		}
		s.raft.BootstrapCluster(configuration)
	} else {
		// send join request to existing node
		msg := []byte(fmt.Sprintf(joinMsg, s.RaftAddr, s.RaftID))
		rsp := tcpRequest(join, msg)
		s.logger.Printf("joining node at %s: %s", join, rsp)
	}
	return nil
}

// start listens for incoming TCP connections
func (s *Store) start(ln net.Listener) {
	for {
		// Accept a connection
		conn, err := ln.Accept()
		if err != nil {
			s.logger.Printf("failed to accept RPC conn: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// handleConn selects correct handler for Raft or KV message
func (s *Store) handleConn(conn net.Conn) {
	// Read the header messaqge
	hdr := make([]byte, 3)
	if _, err := conn.Read(hdr); err != nil {
		if err != io.EOF {
			s.logger.Printf("failed to read byte: %v", err)
		}
		conn.Close()
		return
	}

	// Switch on the header message
	switch {
	case bytes.Equal(hdr, []byte("kv ")):
		s.handleTCP(conn)
	case bytes.Equal(hdr, []byte("rft")):
		s.raftLayer.Handoff(conn)
	default:
		s.logger.Printf("unknown command prefix: %s", string(hdr))
		conn.Write([]byte("ERROR"))
		conn.Close()
	}
}

// WaitLeader polls until leader is known
func (s *Store) WaitLeader() string {
	timeout := time.After(10 * time.Second)
	for {
		if len(s.raft.Leader()) > 0 {
			return string(s.raft.Leader())
		}

		select {
		case <-s.raft.LeaderCh():
			return string(s.raft.Leader())
		case <-time.After(1 * time.Second):
		case <-timeout:
			return ""
		}
	}
}

// Get key from KV Store
func (s *Store) Get(key string) (string, bool) {
	return s.fsm.get(key)
}

// Set adds key to KV Store
func (s *Store) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return s.forward(fmt.Sprintf(setMsg, key, value))
	}

	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	msg, err := json.Marshal(c)
	if err != nil {
		return err
	}
	f := s.raft.Apply(msg, 10*time.Second)

	return f.Error()
}

// Delete removes key from KV Store
func (s *Store) Delete(key string) error {
	if s.raft.State() != raft.Leader {
		return s.forward(fmt.Sprintf(delMsg, key))
	}

	c := &command{
		Op:  "delete",
		Key: key,
	}
	msg, err := json.Marshal(c)
	if err != nil {
		return err
	}
	f := s.raft.Apply(msg, 10*time.Second)

	return f.Error()
}

// Join joins a node, identified by nodeID and located at addr, to this store.
// The node must be ready to respond to Raft communications at that address.
func (s *Store) Join(nodeID, addr string) error {
	if s.raft.State() != raft.Leader {
		return s.forward(fmt.Sprintf(joinMsg, addr, nodeID))
	}

	s.logger.Printf("received join request for remote node %s at %s", nodeID, addr)

	cf := s.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		s.logger.Printf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				s.logger.Printf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	s.logger.Printf("node %s at %s joined successfully", nodeID, addr)
	return nil
}

// Leave removes the node from the cluster
func (s *Store) Leave(nodeID string) error {
	if s.raft.State() != raft.Leader {
		return s.forward(fmt.Sprintf(leaveMsg, s.RaftID))
	}

	s.logger.Printf("received leave request for remote node %s", nodeID)

	cf := s.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		s.logger.Printf("failed to get raft configuration")
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			f := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := f.Error(); err != nil {
				s.logger.Printf("failed to remove server %s, err: %v", nodeID, err)
				return err
			}

			s.logger.Printf("node %s left successfully", nodeID)
			return nil
		}
	}

	s.logger.Printf("node %s not exists in raft group", nodeID)
	return nil
}

func tcpRequest(srvAddr string, message []byte) string {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		return "could not connect to TCP server: " + err.Error()
	}
	defer conn.Close()

	if _, err := conn.Write(message); err != nil {
		return "could not write message to TCP server: " + err.Error()
	}

	buf := make([]byte, rspBuffSize)
	if _, err := conn.Read(buf); err != nil {
		return "could not read response from TCP server: " + err.Error()
	}
	return strings.TrimRight(string(buf), "\x00")
}
