package store

import (
	"errors"
	"fmt"
	"strings"
)

func (s *Store) forwardSet(key, value string) error {
	// find the leader
	leader := s.WaitLeader()
	if len(leader) == 0 {
		return errors.New("no known leader")
	}
	// forward message
	msg := []byte(fmt.Sprintf("kv set %s %s\n", key, value))
	rsp := tcpRequest(leader, msg)
	if !strings.EqualFold(rsp, successMsg) {
		return errors.New(rsp)
	}
	return nil
}

func (s *Store) forwardDel(key string) error {
	// find the leader
	leader := s.WaitLeader()
	if len(leader) == 0 {
		return errors.New("no known leader")
	}
	// forward message
	msg := []byte(fmt.Sprintf("kv del %s\n", key))
	rsp := tcpRequest(leader, msg)
	if !strings.EqualFold(rsp, successMsg) {
		return errors.New(rsp)
	}
	return nil
}
