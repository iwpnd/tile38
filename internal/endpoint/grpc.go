package endpoint

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tidwall/tile38/internal/hservice"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const grpcExpiresAfter = time.Second * 30

// GRPCConn is an endpoint connection
type GRPCConn struct {
	mu    sync.Mutex
	ep    Endpoint
	ex    bool
	t     time.Time
	conn  *grpc.ClientConn
	sconn hservice.HookServiceClient
}

func newGRPCConn(ep Endpoint) *GRPCConn {
	return &GRPCConn{
		ep: ep,
		t:  time.Now(),
	}
}

// Expired returns true if the connection has expired
func (conn *GRPCConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Since(conn.t) > grpcExpiresAfter {
			if conn.conn != nil {
				conn.close()
			}
			conn.ex = true
		}
	}
	return conn.ex
}
func (conn *GRPCConn) close() {
	if conn.conn != nil {
		conn.conn.Close()
		conn.conn = nil
	}
}

// Send sends a message
func (conn *GRPCConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()
	if conn.conn == nil {
		addr := fmt.Sprintf("%s:%d", conn.ep.GRPC.Host, conn.ep.GRPC.Port)
		var err error
		conn.conn, err = grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			conn.close()
			return err
		}
		conn.sconn = hservice.NewHookServiceClient(conn.conn)
	}
	r, err := conn.sconn.Send(context.Background(), &hservice.MessageRequest{Value: msg})
	if err != nil {
		conn.close()
		return err
	}
	if !r.Ok {
		conn.close()
		return errors.New("invalid grpc reply")
	}
	return nil
}
