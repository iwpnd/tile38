package endpoint

import (
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const natsExpiresAfter = time.Second * 30

// NATSConn is an endpoint connection
type NATSConn struct {
	mu   sync.Mutex
	ep   Endpoint
	ex   bool
	t    time.Time
	conn *nats.Conn
}

func newNATSConn(ep Endpoint) *NATSConn {
	return &NATSConn{
		ep: ep,
		t:  time.Now(),
	}
}

// Expired returns true if the connection has expired
func (conn *NATSConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Since(conn.t) > natsExpiresAfter {
			if conn.conn != nil {
				conn.close()
			}
			conn.ex = true
		}
	}
	return conn.ex
}

func (conn *NATSConn) close() {
	if conn.conn != nil {
		conn.conn.Close()
		conn.conn = nil
	}
}

// Send sends a message
func (conn *NATSConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()
	if conn.conn == nil {
		addr := fmt.Sprintf("%s:%d", conn.ep.NATS.Host, conn.ep.NATS.Port)
		var err error
		var opts []nats.Option
		if conn.ep.NATS.User != "" && conn.ep.NATS.Pass != "" {
			opts = append(opts, nats.UserInfo(conn.ep.NATS.User, conn.ep.NATS.Pass))
		}
		if conn.ep.NATS.TLS {
			opts = append(opts, nats.ClientCert(
				conn.ep.NATS.TLSCert, conn.ep.NATS.TLSKey,
			))
		}
		if conn.ep.NATS.Token != "" {
			opts = append(opts, nats.Token(conn.ep.NATS.Token))
		}
		conn.conn, err = nats.Connect(addr, opts...)
		if err != nil {
			conn.close()
			return err
		}
	}
	err := conn.conn.Publish(conn.ep.NATS.Topic, []byte(msg))
	if err != nil {
		conn.close()
		return err
	}

	return nil
}
