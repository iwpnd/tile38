package server

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/resp"
)

type liveMonitorSwitches struct {
	// no fields. everything is managed through the Message
}

func (sub liveMonitorSwitches) Error() string {
	return goingLive
}

func (s *Server) cmdMonitor(msg *Message) (resp.Value, error) {
	if len(msg.Args) != 1 {
		return resp.Value{}, errInvalidNumberOfArguments
	}
	return NOMessage, liveMonitorSwitches{}
}

func (s *Server) liveMonitor(conn net.Conn, rd *PipelineReader, msg *Message) error {
	s.monconnsMu.Lock()
	s.monconns[conn] = true
	s.monconnsMu.Unlock()
	defer func() {
		s.monconnsMu.Lock()
		delete(s.monconns, conn)
		s.monconnsMu.Unlock()
		conn.Close()
	}()
	s.monconnsMu.Lock()
	conn.Write([]byte("+OK\r\n"))
	s.monconnsMu.Unlock()
	msgs, err := rd.ReadMessages()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	for _, msg := range msgs {
		if len(msg.Args) == 1 && strings.ToLower(msg.Args[0]) == "quit" {
			s.monconnsMu.Lock()
			conn.Write([]byte("+OK\r\n"))
			s.monconnsMu.Unlock()
			return nil
		}
	}
	return nil
}

// send messages to live MONITOR clients
func (s *Server) sendMonitor(err error, msg *Message, c *Client, lua bool) {
	s.monconnsMu.RLock()
	n := len(s.monconns)
	s.monconnsMu.RUnlock()
	if n == 0 {
		return
	}
	if (c == nil && !lua) ||
		(err != nil && (err == errInvalidNumberOfArguments ||
			strings.HasPrefix(err.Error(), "unknown command "))) {
		return
	}

	// accept all commands except for these:
	switch strings.ToLower(msg.Command()) {
	case "config", "config set", "config get", "config rewrite",
		"auth", "follow", "slaveof", "replconf",
		"aof", "aofmd5", "client",
		"monitor":
		return
	}

	var line []byte
	for i, arg := range msg.Args {
		if i > 0 {
			line = append(line, ' ')
		}
		line = append(line, strconv.Quote(arg)...)
	}
	tstr := fmt.Sprintf("%.6f", float64(time.Now().UnixNano())/1e9)
	var addr string
	if lua {
		addr = "lua"
	} else {
		addr = c.remoteAddr
	}
	s.monconnsMu.Lock()
	for conn := range s.monconns {
		fmt.Fprintf(conn, "+%s [0 %s] %s\r\n", tstr, addr, line)
	}
	s.monconnsMu.Unlock()
}
