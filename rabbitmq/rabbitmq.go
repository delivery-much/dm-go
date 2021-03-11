package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// SubscribeHandler signature of func to handler/execute in sub message
type SubscribeHandler func(ctx context.Context, msg *Message) error

// Message represents the message/event receive in consumer of RabbitMQ with the body and all fields of delivered message
type Message struct {
	Delivery amqp.Delivery
	Body     []byte
}

// RabbitMQ represents the functions to connect and subribe in RabbitMQ
type RabbitMQ interface {
	Close()
	Ping() error
	Subscribe(cg ConsumerConfig, subHandler SubscribeHandler) error
}

// Client represents the client with connection to RabbitMQ.
type Client struct {
	conn *amqp.Connection
}

// New Connect and returns the AMQP Client that implements the AMQP interface.
func New(amqpURI, projectName string) (RabbitMQ, error) {
	c := &Client{}
	var err error
	cfg := amqp.Config{
		Properties: amqp.Table{
			"connection_name": projectName,
		},
	}
	c.conn, err = amqp.DialConfig(amqpURI, cfg)
	return c, err
}

// ConsumerConfig represents all configs to create and configure a subscribe/consumer
type ConsumerConfig struct {
	ExchangeName string
	ExchangeType string // fanout, topic, direct
	QueueName    string
	BindingKey   string
	ConsumerName string
}

// Subscribe subscribe in a queue in exchange to consume events that is published in her.
// Open a new channel in the client connection.
// Passing the parameters of name of exchange, type of exchange (direct, topic or fanout), queue name, binding key.
// And a handler function to execute in consume, where stay the business logic for execute when the event is received.
func (c *Client) Subscribe(cg ConsumerConfig, subHandler SubscribeHandler) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open a channel: %s", err)
	}

	err = ch.ExchangeDeclare(
		cg.ExchangeName, // name of the exchange
		cg.ExchangeType, // type
		true,            // durable
		false,           // delete when complete
		false,           // internal
		false,           // noWait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("Failed to register an Exchange: %s", err)
	}

	queue, err := ch.QueueDeclare(
		cg.QueueName, // name of the queue
		true,         // durable
		false,        // delete when usused
		false,        // exclusive
		false,        // noWait
		nil,          // arguments
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

	msgs, err := ch.Consume(
		queue.Name,      // queue
		cg.ConsumerName, // tag
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		return fmt.Errorf("Failed to register a consumer: %s", err)
	}

	log.Printf("Consumer registered: exchange %s, queue_name %s, routing_key %s consumer_name %s", cg.ExchangeName, cg.QueueName, cg.BindingKey, cg.ConsumerName)

	go consumeLoop(msgs, subHandler)
	return nil
}

// Close will close the connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Ping get the status of connection with RabbitMQ
func (c *Client) Ping() error {
	if c.conn.IsClosed() {
		return amqp.ErrClosed
	}
	return nil
}

func consumeLoop(deliveries <-chan amqp.Delivery, subHandler SubscribeHandler) {
	for d := range deliveries {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		// Invoke the handlerFunc func we passed as parameter.
		subHandler(ctx, &Message{
			Delivery: d,
			Body:     d.Body,
		})
	}
}
