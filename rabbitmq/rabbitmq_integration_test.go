package rabbitmq

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brokerURI returns the RabbitMQ URI for integration tests, skipping the test when it is not set.
//
//	RABBITMQ_URI=amqp://guest:guest@localhost:5672/ go test ./rabbitmq/ -run Integration
func brokerURI(t *testing.T) string {
	t.Helper()
	uri := os.Getenv("RABBITMQ_URI")
	if uri == "" {
		t.Skip("RABBITMQ_URI not set; skipping RabbitMQ integration test")
	}
	return uri
}

func TestIntegrationSubscribeReceivesPublishedMessage(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.Ping())

	suffix := time.Now().UnixNano()
	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-exchange",
		ExchangeType: "topic",
		QueueName:    "dm-go-test-queue",
		BindingKey:   "dm-go.test",
		ConsumerName: "dm-go-test-consumer",
	}

	received := make(chan *Message, 1)
	err = client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		received <- msg
		return nil
	})
	require.NoError(t, err)

	// publish through an independent channel, as a producer service would
	conn, err := amqp.Dial(uri)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()
	defer ch.QueueDelete(cfg.QueueName, false, false, false)
	defer ch.ExchangeDelete(cfg.ExchangeName, false, false)

	body := []byte("hello from dm-go " + time.Duration(suffix).String())
	err = ch.PublishWithContext(context.Background(), cfg.ExchangeName, cfg.BindingKey, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        body,
	})
	require.NoError(t, err)

	select {
	case msg := <-received:
		assert.Equal(t, body, msg.Body)
		assert.Equal(t, body, msg.Delivery.Body)
		assert.Equal(t, cfg.BindingKey, msg.Delivery.RoutingKey)
		assert.Equal(t, cfg.ExchangeName, msg.Delivery.Exchange)
	case <-time.After(5 * time.Second):
		t.Fatal("message was not delivered to the subscribe handler")
	}
}

func TestIntegrationSubscribeInvalidExchangeType(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	err = client.Subscribe(ConsumerConfig{
		ExchangeName: "dm-go-test-invalid-exchange",
		ExchangeType: "not-a-real-type",
		QueueName:    "dm-go-test-invalid-queue",
		BindingKey:   "dm-go.invalid",
		ConsumerName: "dm-go-test-invalid-consumer",
	}, func(ctx context.Context, msg *Message) error { return nil })

	assert.ErrorContains(t, err, "Failed to register an Exchange")
}

func TestIntegrationPingAfterClose(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	require.NoError(t, client.Ping())

	client.Close()

	assert.ErrorIs(t, client.Ping(), amqp.ErrClosed)
}
