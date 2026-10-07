package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SubscribeHandler signature of func to handler/execute in sub message
type SubscribeHandler func(ctx context.Context, msg *Message) error

// Message represents the message/event receive in consumer of RabbitMQ with the body and all fields of delivered message
type Message struct {
	Delivery amqp.Delivery
	Body     []byte
}

// Publishing is the message to be published. It is an alias of amqp091.Publishing,
// so callers can set Body, ContentType, Headers, DeliveryMode and so on without importing the driver.
type Publishing = amqp.Publishing

// Table is an alias of amqp091.Table, used for exchange, queue and consumer arguments.
type Table = amqp.Table

// Middleware wraps a SubscribeHandler, for example to add tracing, metrics or logging context.
// Middlewares run in the order they are given: the first one is the outermost.
type Middleware func(next SubscribeHandler) SubscribeHandler

// ErrRequeue is the sentinel error that tells a manual-ack consumer to nack the message
// with requeue, so the broker delivers it again. Wrap the cause with Requeue to keep it.
var ErrRequeue = errors.New("rabbitmq: requeue message")

// Requeue wraps err so that a manual-ack consumer nacks the message with requeue.
// errors.Is(Requeue(err), ErrRequeue) and errors.Is(Requeue(err), err) are both true.
func Requeue(err error) error {
	if err == nil {
		return ErrRequeue
	}
	return fmt.Errorf("%w: %w", ErrRequeue, err)
}

// RabbitMQ represents the functions to connect, publish and subscribe in RabbitMQ
type RabbitMQ interface {
	Close()
	Ping() error
	Publish(ctx context.Context, exchange, routingKey string, msg Publishing) error
	Subscribe(cg ConsumerConfig, subHandler SubscribeHandler) error
}

// Default reconnection parameters, applied when the client is created without WithReconnection.
const (
	DefaultReconnectMaxRetries = amqp.DefaultMaxRetryCount
	DefaultReconnectInterval   = amqp.DefaultRetryInterval
)

// ReconnectForever, given to WithReconnection as maxRetries, keeps retrying until the broker is back.
// Use it for long-running consumers that must outlive a broker outage of any length.
const ReconnectForever = math.MaxInt

// Option configures the client created by New.
type Option func(*options)

type options struct {
	recovery    *amqp.Recovery
	middlewares []Middleware
}

func defaultOptions() *options {
	return &options{
		recovery: &amqp.Recovery{
			ReconnectionConfig: &amqp.ReconnectionConfig{
				MaxRetryCount: DefaultReconnectMaxRetries,
				RetryInterval: DefaultReconnectInterval,
			},
		},
	}
}

// WithReconnection sets how many times, and how often, the client tries to reconnect
// after the connection to the broker is lost. Exchanges, queues, bindings and consumers
// registered through Subscribe are re-declared automatically after a successful reconnection.
// maxRetries must be greater than zero, otherwise reconnection is disabled; ReconnectForever
// never gives up.
func WithReconnection(maxRetries int, interval time.Duration) Option {
	return func(o *options) {
		if maxRetries <= 0 {
			o.recovery = nil
			return
		}
		o.recovery = &amqp.Recovery{
			ReconnectionConfig: &amqp.ReconnectionConfig{
				MaxRetryCount: maxRetries,
				RetryInterval: interval,
			},
		}
	}
}

// WithMiddlewares registers middlewares applied to every consumer created by Subscribe.
// They run before the middlewares given in ConsumerConfig.Middlewares.
func WithMiddlewares(mws ...Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, mws...)
	}
}

// WithoutReconnection disables automatic reconnection. When the connection is lost,
// Ping returns an error and consumers stop receiving messages until a new client is created.
func WithoutReconnection() Option {
	return func(o *options) {
		o.recovery = nil
	}
}

// Client represents the client with connection to RabbitMQ.
type Client struct {
	conn        *amqp.Connection
	middlewares []Middleware

	mu        sync.Mutex
	publishCh *amqp.Channel
}

// New Connect and returns the AMQP Client that implements the AMQP interface.
// Automatic reconnection is enabled by default, see WithReconnection and WithoutReconnection.
func New(amqpURI, projectName string, opts ...Option) (RabbitMQ, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	c := &Client{middlewares: o.middlewares}
	var err error
	cfg := amqp.Config{
		Properties: amqp.Table{
			"connection_name": projectName,
		},
		Recovery: o.recovery,
	}
	c.conn, err = amqp.DialConfig(amqpURI, cfg)
	if err != nil {
		return c, err
	}

	if o.recovery != nil {
		states := make(chan *amqp.StateChanged, 8)
		c.conn.NotifyStateChange(states)
		go logStateChanges(projectName, states)
	}
	return c, nil
}

// DefaultHandlerTimeout is the timeout of the context given to the handler when ConsumerConfig.HandlerTimeout is zero.
const DefaultHandlerTimeout = 10 * time.Second

// ConsumerConfig represents all configs to create and configure a subscribe/consumer.
// Only the five name fields are required; the zero value of every other field keeps the
// default behaviour (durable exchange and queue, auto-ack, 10s handler timeout).
type ConsumerConfig struct {
	ExchangeName string
	ExchangeType string // fanout, topic, direct
	QueueName    string
	BindingKey   string
	ConsumerName string

	Exchange ExchangeOptions
	Queue    QueueOptions
	Consume  ConsumeOptions

	// Middlewares wrap the handler of this consumer, after the client-level ones from WithMiddlewares.
	Middlewares []Middleware

	// HandlerTimeout bounds the context given to the handler. Zero means DefaultHandlerTimeout,
	// a negative value disables the timeout.
	HandlerTimeout time.Duration
}

// ExchangeOptions are the optional flags of the exchange declaration.
type ExchangeOptions struct {
	// Durable survives a broker restart. Defaults to true, use NotDurable to set it to false.
	Durable    *bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       Table
}

// QueueOptions are the optional flags of the queue declaration.
type QueueOptions struct {
	// Durable survives a broker restart. Defaults to true, use NotDurable to set it to false.
	Durable    *bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       Table
}

// NotDurable is a ready-made value for ExchangeOptions.Durable and QueueOptions.Durable.
var NotDurable = boolPtr(false)

func boolPtr(b bool) *bool { return &b }

// durable resolves the Durable pointer, defaulting to true when it is nil.
func durable(v *bool) bool {
	return v == nil || *v
}

// ConsumeOptions are the optional flags of the consumer registration.
type ConsumeOptions struct {
	// ManualAck disables auto-ack. The message is then acked when the handler returns nil,
	// nacked with requeue when it returns an error wrapping ErrRequeue (see Requeue),
	// and nacked without requeue (dead-lettered or dropped) on any other error.
	ManualAck bool
	// PrefetchCount limits the unacknowledged messages in flight for this consumer (basic.qos).
	// Zero leaves the broker default (unlimited). Only meaningful with ManualAck.
	PrefetchCount int
	Exclusive     bool
	NoLocal       bool
	NoWait        bool
	Args          Table
}

// Subscribe subscribe in a queue in exchange to consume events that is published in her.
// Open a new channel in the client connection.
// Passing the parameters of name of exchange, type of exchange (direct, topic or fanout), queue name, binding key.
// And a handler function to execute in consume, where stay the business logic for execute when the event is received.
func (c *Client) Subscribe(cg ConsumerConfig, subHandler SubscribeHandler) error {
	if c.conn == nil {
		return amqp.ErrClosed
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open a channel: %s", err)
	}

	err = ch.ExchangeDeclare(
		cg.ExchangeName,              // name of the exchange
		cg.ExchangeType,              // type
		durable(cg.Exchange.Durable), // durable
		cg.Exchange.AutoDelete,       // delete when complete
		cg.Exchange.Internal,         // internal
		cg.Exchange.NoWait,           // noWait
		cg.Exchange.Args,             // arguments
	)
	if err != nil {
		return fmt.Errorf("Failed to register an Exchange: %s", err)
	}

	queue, err := ch.QueueDeclare(
		cg.QueueName,              // name of the queue
		durable(cg.Queue.Durable), // durable
		cg.Queue.AutoDelete,       // delete when usused
		cg.Queue.Exclusive,        // exclusive
		cg.Queue.NoWait,           // noWait
		cg.Queue.Args,             // arguments
	)
	if err != nil {
		return fmt.Errorf("Failed to register an Queue: %s", err)
	}

	err = ch.QueueBind(
		queue.Name,      // name of the queue
		cg.BindingKey,   // bindingKey
		cg.ExchangeName, // sourceExchange
		false,           // noWait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("Queue Bind: %s", err)
	}

	if cg.Consume.PrefetchCount > 0 {
		if err := ch.Qos(cg.Consume.PrefetchCount, 0, false); err != nil {
			return fmt.Errorf("Failed to set prefetch count: %s", err)
		}
	}

	msgs, err := ch.Consume(
		queue.Name,            // queue
		cg.ConsumerName,       // tag
		!cg.Consume.ManualAck, // auto-ack
		cg.Consume.Exclusive,  // exclusive
		cg.Consume.NoLocal,    // no-local
		cg.Consume.NoWait,     // no-wait
		cg.Consume.Args,       // args
	)
	if err != nil {
		return fmt.Errorf("Failed to register a consumer: %s", err)
	}

	log.Printf("Consumer registered: exchange %s, queue_name %s, routing_key %s consumer_name %s", cg.ExchangeName, cg.QueueName, cg.BindingKey, cg.ConsumerName)

	go consumeLoop(msgs, consumer{
		handler:   chain(subHandler, c.middlewares, cg.Middlewares),
		manualAck: cg.Consume.ManualAck,
		timeout:   cg.HandlerTimeout,
		queue:     cg.QueueName,
	})
	return nil
}

// chain wraps handler with the client-level and then the consumer-level middlewares,
// so the first middleware of the client is the outermost one.
func chain(handler SubscribeHandler, groups ...[]Middleware) SubscribeHandler {
	var all []Middleware
	for _, g := range groups {
		all = append(all, g...)
	}
	for i := len(all) - 1; i >= 0; i-- {
		handler = all[i](handler)
	}
	return handler
}

// Publish publishes a message to the exchange with the given routing key.
// The exchange must already exist (for example declared by a Subscribe on the same or another service).
// All publishes share one channel that is opened lazily and reopened when it is found closed.
// While the client is reconnecting the publish fails with amqp091.ErrClosed, callers should retry.
//
// A producer span is created from ctx and its trace context is injected into the message headers
// (traceparent, tracestate, baggage), so the consumer continues the same trace. Without an
// OpenTelemetry provider this is a no-op and the headers are left untouched.
func (c *Client) Publish(ctx context.Context, exchange, routingKey string, msg Publishing) (err error) {
	ctx, span := startPublishSpan(ctx, exchange, routingKey, &msg)
	defer func() { endSpan(span, err) }()

	ch, err := c.publishChannel()
	if err != nil {
		return fmt.Errorf("Failed to open a publish channel: %w", err)
	}

	err = ch.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		msg,
	)
	if err != nil {
		return fmt.Errorf("Failed to publish message to exchange %s with routing key %s: %w", exchange, routingKey, err)
	}
	return nil
}

func (c *Client) publishChannel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.conn.IsClosed() {
		return nil, amqp.ErrClosed
	}
	if c.publishCh != nil && !c.publishCh.IsClosed() {
		return c.publishCh, nil
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}
	c.publishCh = ch
	return ch, nil
}

// Close will close the publish channel and the connection.
func (c *Client) Close() {
	c.mu.Lock()
	if c.publishCh != nil {
		c.publishCh.Close()
		c.publishCh = nil
	}
	c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
}

// Ping get the status of connection with RabbitMQ
func (c *Client) Ping() error {
	if c.conn == nil || c.conn.IsClosed() {
		return amqp.ErrClosed
	}
	return nil
}

type consumer struct {
	handler   SubscribeHandler
	manualAck bool
	timeout   time.Duration
	queue     string
}

func consumeLoop(deliveries <-chan amqp.Delivery, c consumer) {
	for d := range deliveries {
		handleDelivery(d, c)
	}
}

func handleDelivery(d amqp.Delivery, c consumer) {
	// The consumer span continues the trace propagated in the message headers by Publish.
	ctx, span := startConsumeSpan(context.Background(), c.queue, d)
	switch {
	case c.timeout == 0:
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultHandlerTimeout)
		defer cancel()
	case c.timeout > 0:
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	// Invoke the handlerFunc func we passed as parameter.
	err := c.handler(ctx, &Message{
		Delivery: d,
		Body:     d.Body,
	})
	defer endSpan(span, err)
	if !c.manualAck {
		return
	}

	switch {
	case err == nil:
		if ackErr := d.Ack(false); ackErr != nil {
			log.Printf("RabbitMQ ack failed: queue %s, routing_key %s, delivery_tag %d: %v", c.queue, d.RoutingKey, d.DeliveryTag, ackErr)
		}
	case errors.Is(err, ErrRequeue):
		log.Printf("RabbitMQ message requeued: queue %s, routing_key %s, delivery_tag %d: %v", c.queue, d.RoutingKey, d.DeliveryTag, err)
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.Printf("RabbitMQ requeue failed: queue %s, routing_key %s, delivery_tag %d: %v", c.queue, d.RoutingKey, d.DeliveryTag, nackErr)
		}
	default:
		log.Printf("RabbitMQ message rejected: queue %s, routing_key %s, delivery_tag %d: %v", c.queue, d.RoutingKey, d.DeliveryTag, err)
		if nackErr := d.Nack(false, false); nackErr != nil {
			log.Printf("RabbitMQ reject failed: queue %s, routing_key %s, delivery_tag %d: %v", c.queue, d.RoutingKey, d.DeliveryTag, nackErr)
		}
	}
}

// logStateChanges logs the connection lifecycle (reconnecting, recovered, closed) reported by the driver.
func logStateChanges(projectName string, states <-chan *amqp.StateChanged) {
	for s := range states {
		switch {
		case s.To == amqp.StateReconnecting:
			log.Printf("RabbitMQ connection %s lost, reconnecting", projectName)
		case s.From == amqp.StateReconnecting && s.To == amqp.StateOpen:
			log.Printf("RabbitMQ connection %s recovered", projectName)
			for _, e := range s.SkippedTopologyEntities {
				log.Printf("RabbitMQ connection %s could not recover %s: %v", projectName, e.EntityType, e.Err)
			}
		case s.To == amqp.StateClosed && s.Err != nil:
			log.Printf("RabbitMQ connection %s closed: %v", projectName, s.Err)
		}
	}
}
