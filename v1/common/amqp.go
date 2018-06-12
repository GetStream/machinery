package common

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// AMQPConnector ...
type AMQPConnector struct {
	connManager *amqpConnectionManager

	exchangeMaxRetries   int
	exchangeRetryTimeout time.Duration
}

type AMQPConnectorOption func(c *AMQPConnector)

func WithAMQPExchangeMaxRetries(retries int) AMQPConnectorOption {
	return func(c *AMQPConnector) {
		c.exchangeMaxRetries = retries
	}
}

func WithAMQPExchangeRetryTimeout(timeout time.Duration) AMQPConnectorOption {
	return func(c *AMQPConnector) {
		c.exchangeRetryTimeout = timeout
	}
}

func WithAMQPConnectionMaxRetries(retries int) AMQPConnectorOption {
	return func(c *AMQPConnector) {
		c.connManager.connectionMaxRetries = retries
	}
}

func WithAMQPConnectionRetryTimeout(timeout time.Duration) AMQPConnectorOption {
	return func(c *AMQPConnector) {
		c.connManager.connectionRetryTimeout = timeout
	}
}

func NewAMQPConnector(url string, tlsConfig *tls.Config, opts ...AMQPConnectorOption) *AMQPConnector {
	c := &AMQPConnector{
		connManager: newAMQPConnectionManager(url, tlsConfig),

		exchangeMaxRetries:   3,
		exchangeRetryTimeout: time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type wrappedError struct {
	info string
	err  error
}

func (e wrappedError) Error() string {
	return fmt.Sprintf("%s: %s", e.info, e.err)
}

func wrapError(info string, err error) error {
	return wrappedError{
		info: info,
		err:  err,
	}
}

// Exchange declares an exchange, opens a channel declares and binds the queue and enables publish notifications using the existing RabbitMQ connection.
func (ac *AMQPConnector) Exchange(exchange, exchangeType, queueName string, queueDurable, queueDelete bool, queueBindingKey string, exchangeDeclareArgs, queueDeclareArgs, queueBindingArgs amqp.Table) (*amqp.Channel, amqp.Queue, <-chan amqp.Confirmation, error) {
	var lastErr error
	for retry := 0; retry < ac.exchangeMaxRetries; retry++ {
		channel, queue, confirmChan, err := ac.exchange(exchange, exchangeType, queueName, queueDurable, queueDelete, queueBindingKey, exchangeDeclareArgs, queueDeclareArgs, queueBindingArgs)
		if err != nil {
			lastErr = err
			if wrapped, ok := err.(wrappedError); ok {
				if wrapped.err == amqp.ErrClosed {
					ac.connManager.reset()
				}
			}
			time.Sleep(ac.exchangeRetryTimeout)
			continue
		}
		return channel, queue, confirmChan, nil
	}
	return nil, amqp.Queue{}, nil, wrapError("too many retries", lastErr)
}

func (ac *AMQPConnector) exchange(exchange, exchangeType, queueName string, queueDurable, queueDelete bool, queueBindingKey string, exchangeDeclareArgs, queueDeclareArgs, queueBindingArgs amqp.Table) (*amqp.Channel, amqp.Queue, <-chan amqp.Confirmation, error) {
	channel, err := ac.Channel()
	if err != nil {
		return nil, amqp.Queue{}, nil, err
	}

	if exchange != "" {
		// Declare an exchange
		err := channel.ExchangeDeclare(
			exchange,            // name of the exchange
			exchangeType,        // type
			true,                // durable
			false,               // delete when complete
			false,               // internal
			false,               // noWait
			exchangeDeclareArgs, // arguments
		)
		if err != nil {
			channel.Close()
			return nil, amqp.Queue{}, nil, wrapError("exchange declare error", err)
		}
	}

	var queue amqp.Queue
	if queueName != "" {
		// Declare a queue
		queue, err = channel.QueueDeclare(
			queueName,        // name
			queueDurable,     // durable
			queueDelete,      // delete when unused
			false,            // exclusive
			false,            // no-wait
			queueDeclareArgs, // arguments
		)
		if err != nil {
			channel.Close()
			return nil, amqp.Queue{}, nil, wrapError("queue declare error", err)
		}

		// Bind the queue
		err = channel.QueueBind(
			queue.Name,       // name of the queue
			queueBindingKey,  // binding key
			exchange,         // source exchange
			false,            // noWait
			queueBindingArgs, // arguments
		)
		if err != nil {
			channel.Close()
			return nil, amqp.Queue{}, nil, wrapError("queue bind error", err)
		}
	}

	// Enable publish confirmations
	if err := channel.Confirm(false); err != nil {
		channel.Close()
		return nil, amqp.Queue{}, nil, wrapError("channel could not be put into confirm mode", err)
	}

	return channel, queue, channel.NotifyPublish(make(chan amqp.Confirmation, 1)), nil
}

// DeleteQueue deletes a queue by name
func (ac *AMQPConnector) DeleteQueue(channel *amqp.Channel, queueName string) error {
	// First return value is number of messages removed
	_, err := channel.QueueDelete(
		queueName, // name
		false,     // ifUnused
		false,     // ifEmpty
		false,     // noWait
	)

	return err
}

// InspectQueue provides information about a specific queue
func (*AMQPConnector) InspectQueue(channel *amqp.Channel, queueName string) (*amqp.Queue, error) {
	queueState, err := channel.QueueInspect(queueName)
	if err != nil {
		return nil, fmt.Errorf("Queue inspect error: %s", err)
	}
	return &queueState, nil
}

func (ac *AMQPConnector) Channel() (*amqp.Channel, error) {
	conn, err := ac.connManager.get()
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Open channel error: %s", err)
	}
	return channel, nil
}

func (ac *AMQPConnector) ErrChan() chan error {
	return ac.connManager.errChan
}

type amqpConnectionManager struct {
	url       string
	tlsConfig *tls.Config
	conn      *amqp.Connection
	connChan  chan struct{}
	errChan   chan error
	mu        *sync.Mutex

	connectionRetryTimeout time.Duration
	connectionMaxRetries   int
}

func newAMQPConnectionManager(url string, tlsConfig *tls.Config) *amqpConnectionManager {
	m := &amqpConnectionManager{
		url:       url,
		tlsConfig: tlsConfig,
		errChan:   make(chan error),
		connChan:  make(chan struct{}),
		mu:        &sync.Mutex{},
		connectionRetryTimeout: 5 * time.Second,
		connectionMaxRetries:   3,
	}
	close(m.connChan)
	return m
}

func (m *amqpConnectionManager) get() (*amqp.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.connChan:
		// create a new connection and store it
		conn, err := m.connect()
		if err != nil {
			return nil, err
		}
		m.conn = conn
		m.connChan = make(chan struct{})
		// intercept close events
		m.notifyClose()
	default:
	}
	return m.conn, nil
}

func (m *amqpConnectionManager) connect() (conn *amqp.Connection, err error) {
	for retry := 0; retry < m.connectionMaxRetries; retry++ {
		conn, err = amqp.DialTLS(m.url, m.tlsConfig)
		if err == nil {
			return conn, nil
		}
		time.Sleep(m.connectionRetryTimeout)
	}
	return nil, wrapError("too many retries", err)
}

func (m *amqpConnectionManager) notifyClose() {
	closeChan := m.conn.NotifyClose(make(chan *amqp.Error, 1))
	go func() {
		select {
		case <-m.connChan:
			return
		case err := <-closeChan:
			m.errChan <- err
			m.closeConnChan()
		}
	}()
}

func (m *amqpConnectionManager) closeConnChan() { // do not close the connection channel if already closed
	select {
	case <-m.connChan:
	default:
		close(m.connChan)
	}
}

func (m *amqpConnectionManager) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return
	}

	m.closeConnChan()
	m.conn.Close()
	m.conn = nil
}
