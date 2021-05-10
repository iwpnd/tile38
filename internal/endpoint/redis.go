package endpoint

import (
	"fmt"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
)

const redisExpiresAfter = time.Second * 30

// RedisConn is an endpoint connection
type RedisConn struct {
	mu   sync.Mutex
	ep   Endpoint
	ex   bool
	t    time.Time
	conn redis.Conn
}

func newRedisConn(ep Endpoint) *RedisConn {
	return &RedisConn{
		ep: ep,
		t:  time.Now(),
	}
}

// Expired returns true if the connection has expired
func (conn *RedisConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Since(conn.t) > redisExpiresAfter {
			if conn.conn != nil {
				conn.close()
			}
			conn.ex = true
		}
	}
	return conn.ex
}

func (conn *RedisConn) close() {
	if conn.conn != nil {
		conn.conn.Close()
		conn.conn = nil
	}
}

// Send sends a message
func (conn *RedisConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()
	if conn.conn == nil {
		addr := fmt.Sprintf("%s:%d", conn.ep.Redis.Host, conn.ep.Redis.Port)
		var err error
		conn.conn, err = redis.Dial("tcp", addr)
		if err != nil {
			conn.close()
			return err
		}
	}
	_, err := redis.Int(conn.conn.Do("PUBLISH", conn.ep.Redis.Channel, msg))
	if err != nil {
		conn.close()
		return err
	}
	return nil
}
