package endpoint

import (
	"cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"google.golang.org/api/option"
	"sync"
	"time"
)

const pubsubExpiresAfter = time.Second * 30

var errMissingGoogleCredentials = errors.New("Could not find GCP credentials - no credential path was submitted and envar GOOGLE_APPLICATION_CREDENTIALS is not set")

// SQSConn is an endpoint connection
type PubSubConn struct {
	mu      sync.Mutex
	ep      Endpoint
	svc     *pubsub.Client
	topic   *pubsub.Topic
	channel *amqp.Channel
	ex      bool
	t       time.Time
}

func (conn *PubSubConn) generatePubSubURL() string {
	if conn.ep.SQS.PlainURL != "" {
		return conn.ep.SQS.PlainURL
	}
	return "projects/" + conn.ep.PubSub.Project + "/topics/" + conn.ep.PubSub.Topic
}

func (conn *PubSubConn) close() {
	if conn.svc != nil {
		conn.svc.Close()
		conn.svc = nil
	}
}

// Send sends a message
func (conn *PubSubConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.ex {
		return errExpired
	}

	ctx := context.Background()

	conn.t = time.Now()

	if conn.svc == nil {
		var creds option.ClientOption
		var svc *pubsub.Client
		var err error
		credPath := conn.ep.PubSub.CredPath

		if credPath != "" {
			creds = option.WithCredentialsFile(credPath)
			svc, err = pubsub.NewClient(ctx, conn.ep.PubSub.Project, creds)
		} else {
			svc, err = pubsub.NewClient(ctx, conn.ep.PubSub.Project)
		}

		if err != nil {
			fmt.Println(err)
			return err
		}

		topic := svc.Topic(conn.ep.PubSub.Topic)

		conn.svc = svc
		conn.topic = topic
	}

	// Send message
	res := conn.topic.Publish(ctx, &pubsub.Message{
		Data: []byte(msg),
	})
	_, err := res.Get(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (conn *PubSubConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Now().Sub(conn.t) > pubsubExpiresAfter {
			conn.ex = true
			conn.close()
		}
	}
	return conn.ex
}

func newPubSubConn(ep Endpoint) *PubSubConn {
	return &PubSubConn{
		ep: ep,
		t:  time.Now(),
	}
}
