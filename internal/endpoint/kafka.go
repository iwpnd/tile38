package endpoint

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	lg "log"

	"github.com/Shopify/sarama"
	"github.com/tidwall/gjson"
	"github.com/tidwall/tile38/internal/log"
)

const kafkaExpiresAfter = time.Second * 30

// KafkaConn is an endpoint connection
type KafkaConn struct {
	mu   sync.Mutex
	ep   Endpoint
	conn sarama.SyncProducer
	ex   bool
	t    time.Time
}

// Expired returns true if the connection has expired
func (conn *KafkaConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Since(conn.t) > kafkaExpiresAfter {
			if conn.conn != nil {
				conn.close()
			}
			conn.ex = true
		}
	}
	return conn.ex
}

func (conn *KafkaConn) close() {
	if conn.conn != nil {
		conn.conn.Close()
		conn.conn = nil
	}
}

// Send sends a message
func (conn *KafkaConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()

	if log.Level > 2 {
		sarama.Logger = lg.New(log.Output(), "[sarama] ", 0)
	}

	uri := fmt.Sprintf("%s:%d", conn.ep.Kafka.Host, conn.ep.Kafka.Port)
	if conn.conn == nil {
		cfg := sarama.NewConfig()

		if conn.ep.Kafka.TLS {
			log.Debugf("building kafka tls config")
			tlsConfig, err := newKafkaTLSConfig(conn.ep.Kafka.CertFile, conn.ep.Kafka.KeyFile, conn.ep.Kafka.CACertFile)
			if err != nil {
				return err
			}
			cfg.Net.TLS.Enable = true
			cfg.Net.TLS.Config = tlsConfig
		}

		cfg.Net.DialTimeout = time.Second
		cfg.Net.ReadTimeout = time.Second * 5
		cfg.Net.WriteTimeout = time.Second * 5
		// Fix #333 : fix backward incompatibility introduced by sarama library
		cfg.Producer.Return.Successes = true
		cfg.Version = sarama.V0_10_0_0

		c, err := sarama.NewSyncProducer([]string{uri}, cfg)
		if err != nil {
			return err
		}

		conn.conn = c
	}

	// parse json again to get out info for our kafka key
	key := gjson.Get(msg, "key")
	id := gjson.Get(msg, "id")
	keyValue := fmt.Sprintf("%s-%s", key.String(), id.String())

	message := &sarama.ProducerMessage{
		Topic: conn.ep.Kafka.TopicName,
		Key:   sarama.StringEncoder(keyValue),
		Value: sarama.StringEncoder(msg),
	}

	_, offset, err := conn.conn.SendMessage(message)
	if err != nil {
		conn.close()
		return err
	}

	if offset < 0 {
		conn.close()
		return errors.New("invalid kafka reply")
	}

	return nil
}

func newKafkaConn(ep Endpoint) *KafkaConn {
	return &KafkaConn{
		ep: ep,
		t:  time.Now(),
	}
}

func newKafkaTLSConfig(CertFile, KeyFile, CACertFile string) (*tls.Config, error) {
	tlsConfig := tls.Config{}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(CertFile, KeyFile)
	if err != nil {
		return &tlsConfig, err
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Load CA cert
	caCert, err := ioutil.ReadFile(CACertFile)
	if err != nil {
		return &tlsConfig, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig.RootCAs = caCertPool

	return &tlsConfig, err
}
