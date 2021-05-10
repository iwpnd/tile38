package endpoint

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/tidwall/tile38/internal/log"
)

const sqsExpiresAfter = time.Second * 30

// SQSConn is an endpoint connection
type SQSConn struct {
	mu      sync.Mutex
	ep      Endpoint
	session *session.Session
	svc     *sqs.SQS
	ex      bool
	t       time.Time
}

func (conn *SQSConn) generateSQSURL() string {
	if conn.ep.SQS.PlainURL != "" {
		return conn.ep.SQS.PlainURL
	}
	return "https://sqs." + conn.ep.SQS.Region + ".amazonaws.com/" +
		conn.ep.SQS.QueueID + "/" + conn.ep.SQS.QueueName
}

// Expired returns true if the connection has expired
func (conn *SQSConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Since(conn.t) > sqsExpiresAfter {
			conn.ex = true
			conn.close()
		}
	}
	return conn.ex
}

func (conn *SQSConn) close() {
	if conn.svc != nil {
		conn.svc = nil
		conn.session = nil
	}
}

// Send sends a message
func (conn *SQSConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()

	if conn.svc == nil && conn.session == nil {
		var creds *credentials.Credentials
		credPath := conn.ep.SQS.CredPath
		if credPath != "" {
			credProfile := conn.ep.SQS.CredProfile
			if credProfile == "" {
				credProfile = "default"
			}
			creds = credentials.NewSharedCredentials(credPath, credProfile)
		}
		var region string
		if conn.ep.SQS.Region != "" {
			region = conn.ep.SQS.Region
		} else {
			region = sqsRegionFromPlainURL(conn.ep.SQS.PlainURL)
		}
		sess := session.Must(session.NewSession(&aws.Config{
			Region:                        &region,
			Credentials:                   creds,
			CredentialsChainVerboseErrors: aws.Bool(log.Level >= 3),
			MaxRetries:                    aws.Int(5),
		}))
		svc := sqs.New(sess)
		if conn.ep.SQS.CreateQueue {
			svc.CreateQueue(&sqs.CreateQueueInput{
				QueueName: aws.String(conn.ep.SQS.QueueName),
				Attributes: map[string]*string{
					"DelaySeconds":           aws.String("60"),
					"MessageRetentionPeriod": aws.String("86400"),
				},
			})
		}
		conn.session = sess
		conn.svc = svc
	}

	queueURL := conn.generateSQSURL()
	// Send message
	sendParams := &sqs.SendMessageInput{
		MessageBody: aws.String(msg),
		QueueUrl:    aws.String(queueURL),
	}
	_, err := conn.svc.SendMessage(sendParams)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func newSQSConn(ep Endpoint) *SQSConn {
	return &SQSConn{
		ep: ep,
		t:  time.Now(),
	}
}

func probeSQS(s string) bool {
	// https://sqs.eu-central-1.amazonaws.com/123456789/myqueue
	return strings.HasPrefix(s, "https://sqs.") &&
		strings.Contains(s, ".amazonaws.com/")
}

func sqsRegionFromPlainURL(s string) string {
	parts := strings.Split(s, "https://sqs.")
	if len(parts) > 1 {
		parts = strings.Split(parts[1], ".amazonaws.com/")
		if len(parts) > 1 {
			return parts[0]
		}
	}
	return ""
}
