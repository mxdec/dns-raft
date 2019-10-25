package store

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// raftLayer taken from:
// https://github.com/hashicorp/consul/blob/master/agent/consul/raft_rpc.go
// we can not import this file alone, so we have to copy it.
type raftLayer struct {
	// addr is the listener address to return.
	addr net.Addr

	// connCh is used to accept connections.
	connCh chan net.Conn

	// Tracks if we are closed
	closed    bool
	closeCh   chan struct{}
	closeLock sync.Mutex
}

func newRaftLayer(addr *net.TCPAddr) *raftLayer {
	return &raftLayer{
		addr:    addr,
		connCh:  make(chan net.Conn),
		closeCh: make(chan struct{}),
	}
}

// Handoff is used to hand off a connection to the
// raftLayer. This allows it to be Accept()'ed
func (l *raftLayer) Handoff(c net.Conn) error {
	select {
	case l.connCh <- c:
		return nil
	case <-l.closeCh:
		return fmt.Errorf("Raft layer closed")
	}
}

// Accept is used to return connection which are
// dialed to be used with the Raft layer
func (l *raftLayer) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connCh:
		return conn, nil
	case <-l.closeCh:
		return nil, fmt.Errorf("Raft layer closed")
	}
}

// Close is used to stop listening for Raft connections
func (l *raftLayer) Close() error {
	l.closeLock.Lock()
	defer l.closeLock.Unlock()

	if !l.closed {
		l.closed = true
		close(l.closeCh)
	}
	return nil
}

// Addr is used to return the address of the listener
func (l *raftLayer) Addr() net.Addr {
	return l.addr
}

// Dial is used to create a new outgoing connection
func (l *raftLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	d := &net.Dialer{
		Timeout: timeout,
	}
	conn, err := d.Dial("tcp", string(address))
	if err != nil {
		return nil, err
	}

	// Write the Raft header to set the mode
	_, err = conn.Write([]byte("rft"))
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, err
}
