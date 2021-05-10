package endpoint

import (
	"fmt"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tidwall/tile38/internal/log"
)

const disqueExpiresAfter = time.Second * 30

// DisqueConn is an endpoint connection
type DisqueConn struct {
	mu   sync.Mutex
	ep   Endpoint
	ex   bool
	t    time.Time
	conn redis.Conn
}

func newDisqueConn(ep Endpoint) *DisqueConn {
	return &DisqueConn{
		ep: ep,
		t:  time.Now(),
	}
}

// Expired returns true if the connection has expired
func (conn *DisqueConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Since(conn.t) > disqueExpiresAfter {
			if conn.conn != nil {
				conn.close()
			}
			conn.ex = true
		}
	}
	return conn.ex
}

func (conn *DisqueConn) close() {
	if conn.conn != nil {
		conn.conn.Close()
		conn.conn = nil
	}
}

// Send sends a message
func (conn *DisqueConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()
	if conn.conn == nil {
		addr := fmt.Sprintf("%s:%d", conn.ep.Disque.Host, conn.ep.Disque.Port)
		var err error
		conn.conn, err = redis.Dial("tcp", addr)
		if err != nil {
			return err
		}
	}

	var args []interface{}
	args = append(args, conn.ep.Disque.QueueName, msg, 0)
	if conn.ep.Disque.Options.Replicate > 0 {
		args = append(args, "REPLICATE", conn.ep.Disque.Options.Replicate)
	}

	reply, err := redis.String(conn.conn.Do("ADDJOB", args...))
	if err != nil {
		conn.close()
		return err
	}
	log.Debugf("Disque: ADDJOB '%s'", reply)
	return nil
}
