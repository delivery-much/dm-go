package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInvalidURI(t *testing.T) {
	client, err := New("not-a-valid-uri", "dm-go-test")

	assert.Error(t, err)
	assert.NotNil(t, client)
}

func TestNewUnreachableBroker(t *testing.T) {
	// port 1 is reserved and never has a broker listening, so the dial fails fast
	client, err := New("amqp://guest:guest@127.0.0.1:1/", "dm-go-test")

	require.Error(t, err)
	require.NotNil(t, client)
	assert.Nil(t, client.(*Client).conn)
}

func TestCloseNilConnection(t *testing.T) {
	client := &Client{}

	assert.NotPanics(t, client.Close)
}

func TestMessageDeliveryType(t *testing.T) {
	msg := &Message{
		Delivery: amqp.Delivery{Body: []byte("payload")},
		Body:     []byte("payload"),
	}

	assert.Equal(t, msg.Body, msg.Delivery.Body)
}

func TestConsumeLoopDeliversEveryMessage(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	var mu sync.Mutex
	var received []string

	done := make(chan struct{})
	go func() {
		consumeLoop(deliveries, func(ctx context.Context, msg *Message) error {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, string(msg.Body))
			return nil
		})
		close(done)
	}()

	for _, body := range []string{"one", "two", "three"} {
		deliveries <- amqp.Delivery{Body: []byte(body)}
	}
	close(deliveries)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consumeLoop did not return after the deliveries channel was closed")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"one", "two", "three"}, received)
}

func TestConsumeLoopPassesDeliveryAndTimeoutContext(t *testing.T) {
	deliveries := make(chan amqp.Delivery, 1)
	var got *Message
	var deadline time.Time
	var hasDeadline bool

	deliveries <- amqp.Delivery{
		Body:       []byte("payload"),
		RoutingKey: "orders.created",
		MessageId:  "msg-1",
	}
	close(deliveries)

	consumeLoop(deliveries, func(ctx context.Context, msg *Message) error {
		got = msg
		deadline, hasDeadline = ctx.Deadline()
		return nil
	})

	require.NotNil(t, got)
	assert.Equal(t, []byte("payload"), got.Body)
	assert.Equal(t, []byte("payload"), got.Delivery.Body)
	assert.Equal(t, "orders.created", got.Delivery.RoutingKey)
	assert.Equal(t, "msg-1", got.Delivery.MessageId)

	require.True(t, hasDeadline, "handler context must carry a timeout")
	assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, 2*time.Second)
}

func TestConsumeLoopKeepsRunningAfterHandlerError(t *testing.T) {
	deliveries := make(chan amqp.Delivery, 2)
	deliveries <- amqp.Delivery{Body: []byte("fails")}
	deliveries <- amqp.Delivery{Body: []byte("succeeds")}
	close(deliveries)

	var calls int
	consumeLoop(deliveries, func(ctx context.Context, msg *Message) error {
		calls++
		if string(msg.Body) == "fails" {
			return errors.New("handler error")
		}
		return nil
	})

	assert.Equal(t, 2, calls, "a handler error must not stop the consume loop")
}
