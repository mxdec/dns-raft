package store

import (
	"errors"
	"fmt"
	"strings"
)

func (s *Store) forwardSet(key, value string) error {
	// find the leader
	leader := string(s.raft.Leader())
	if len(leader) == 0 {
		return errors.New("no known leader")
	}
	// forward message
	msg := []byte(fmt.Sprintf("kv set %s %s\n", key, value))
	rsp := tcpRequest(leader, msg)
	s.logger.Printf("set command forwarded to %s: %s", leader, rsp)
	if !strings.EqualFold(rsp, successMsg) {
		return errors.New(rsp)
	}
	return nil
}

func (s *Store) forwardDel(key string) error {
	// find the leader
	leader := string(s.raft.Leader())
	if len(leader) == 0 {
		return errors.New("no known leader")
	}
	// forward message
	msg := []byte(fmt.Sprintf("kv del %s\n", key))
	rsp := tcpRequest(leader, msg)
	s.logger.Printf("del command forwarded to %s", leader)
	if !strings.EqualFold(rsp, successMsg) {
		return errors.New(rsp)
	}
	return nil
}
