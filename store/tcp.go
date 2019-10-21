package store

import (
	"bufio"
	"net"
	"strings"
)

func (s *Store) handleTCP(conn net.Conn) {
	defer conn.Close()
	input, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		s.logger.Println("error reading:", err.Error())
		return
	}
	tmp := strings.TrimSpace(string(input))

	// trim spaces
	cmd := strings.SplitN(tmp, " ", 3)
	// handle command
	rsp := s.handleCmd(cmd)
	s.logger.Printf("tcp cmd %s: %s\n", cmd[0], rsp)
	// send a response back
	conn.Write([]byte(rsp))
}

// Select the handler.
func (s *Store) handleCmd(cmd []string) string {
	if len(cmd) == 0 {
		return errorMsg
	}
	verb := strings.ToLower(cmd[0])
	args := cmd[1:]

	switch verb {
	case "ping":
		return "PONG"
	case "join":
		return s.handleJoin(args)
	case "leave":
		return s.handleLeave(args)
	case "get":
		return s.handleGet(args)
	case "set":
		return s.handleSet(args)
	case "del":
		return s.handleDel(args)
	default:
		return errorMsg
	}
}

func (s *Store) handleJoin(args []string) string {
	if len(args) != 2 {
		return errorMsg
	}

	raftAddr := args[0]
	nodeID := args[1]
	if err := s.Join(nodeID, raftAddr); err != nil {
		return err.Error()
	}
	return successMsg
}

func (s *Store) handleLeave(args []string) string {
	if len(args) != 1 {
		return errorMsg
	}

	nodeID := args[0]
	if err := s.Leave(nodeID); err != nil {
		return err.Error()
	}
	return successMsg
}

func (s *Store) handleGet(args []string) string {
	if len(args) != 1 {
		return errorMsg
	}

	k := args[0]
	v, ok := s.Get(k)
	if !ok {
		return errorMsg
	}
	return v
}

func (s *Store) handleSet(args []string) string {
	if len(args) != 2 {
		return errorMsg
	}

	k := args[0]
	v := args[1]
	if err := s.Set(k, v); err != nil {
		return err.Error()
	}
	return successMsg
}

func (s *Store) handleDel(args []string) string {
	if len(args) != 1 {
		return errorMsg
	}

	k := args[0]
	if err := s.Delete(k); err != nil {
		return err.Error()
	}
	return successMsg
}
