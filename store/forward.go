package store

import (
	"errors"
	"strings"
)

func (s *Store) forward(msg string) error {
	// find the leader
	leader := s.WaitLeader()
	if len(leader) == 0 {
		return errors.New("no known leader")
	}
	// forward message
	rsp := tcpRequest(leader, []byte(msg))
	if !strings.EqualFold(rsp, successMsg) {
		return errors.New(rsp)
	}
	return nil
}
