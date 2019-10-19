package store

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// Store is a simple key-value store, where all changes are made via Raft consensus.
type Store struct {
	RaftAddr string
	RaftID   string

	mu sync.Mutex
	m  map[string]string // The key-value store for the system.

	raft *raft.Raft // The consensus mechanism

	logger *log.Logger
}

type command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// InitStore returns an initialized KV store
func InitStore(addr, join, id string) *Store {
	var store Store

	store.RaftAddr = addr
	store.RaftID = id
	store.m = make(map[string]string)
	store.logger = log.New(os.Stderr, "", log.LstdFlags)

	// start Raft Listener
	if err := store.Open(join); err != nil {
		store.logger.Println(err.Error())
		panic(err.Error())
	}
	return &store
}

// Open initializes the store
func (s *Store) Open(join string) error {
	// Setup Raft configuration.
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(s.RaftID)

	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", s.RaftAddr)
	if err != nil {
		return err
	}

	trans, err := raft.NewTCPTransport(s.RaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	// Create the snapshot store, log store and stable store in memory
	snap := raft.NewInmemSnapshotStore()
	log := raft.NewInmemStore()
	stable := raft.NewInmemStore()

	// Instantiate the Raft systems
	r, err := raft.NewRaft(config, (*fsm)(s), log, stable, snap, trans)
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
		rsp := tcpRequest(join, fmt.Sprintf("join %s %s\n", s.RaftAddr, s.RaftID))
		s.logger.Printf("joining node at %s: %s", join, rsp)
	}
	return nil
}

// Leader is used to return the current leader of the cluster
func (s *Store) Leader() raft.ServerAddress {
	return s.raft.Leader()
}

// IsLeader is used to return the current leader of the cluster
func (s *Store) IsLeader() bool {
	return s.raft.State() == raft.Leader
}

// LeaderCh https://github.com/hashicorp/raft/blob/master/api.go#L945
func (s *Store) LeaderCh() <-chan bool {
	return s.raft.LeaderCh()
}

// Get key from KV Store
func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[key]
	return v, ok
}

// Set adds key to KV Store
func (s *Store) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
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
		return fmt.Errorf("not leader")
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

			s.logger.Printf("node %s leaved successfully", nodeID)
			return nil
		}
	}

	s.logger.Printf("node %s not exists in raft group", nodeID)
	return nil
}

func tcpRequest(srvAddr, message string) string {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		return "could not connect to TCP server: " + err.Error()
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(message)); err != nil {
		return "could not write message to TCP server: " + err.Error()
	}

	out := make([]byte, 1024)
	if _, err := conn.Read(out); err != nil {
		return "could not read response from TCP server: " + err.Error()
	}
	return string(out)
}
