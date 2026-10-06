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
		consumeLoop(deliveries, consumer{handler: func(ctx context.Context, msg *Message) error {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, string(msg.Body))
			return nil
		}})
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

	consumeLoop(deliveries, consumer{handler: func(ctx context.Context, msg *Message) error {
		got = msg
		deadline, hasDeadline = ctx.Deadline()
		return nil
	}})

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
	consumeLoop(deliveries, consumer{handler: func(ctx context.Context, msg *Message) error {
		calls++
		if string(msg.Body) == "fails" {
			return errors.New("handler error")
		}
		return nil
	}})

	assert.Equal(t, 2, calls, "a handler error must not stop the consume loop")
}

func TestDefaultOptionsEnableReconnection(t *testing.T) {
	o := defaultOptions()

	require.NotNil(t, o.recovery)
	require.NotNil(t, o.recovery.ReconnectionConfig)
	assert.Equal(t, DefaultReconnectMaxRetries, o.recovery.ReconnectionConfig.MaxRetryCount)
	assert.Equal(t, DefaultReconnectInterval, o.recovery.ReconnectionConfig.RetryInterval)
}

func TestWithReconnection(t *testing.T) {
	o := defaultOptions()
	WithReconnection(3, 250*time.Millisecond)(o)

	require.NotNil(t, o.recovery)
	assert.Equal(t, 3, o.recovery.ReconnectionConfig.MaxRetryCount)
	assert.Equal(t, 250*time.Millisecond, o.recovery.ReconnectionConfig.RetryInterval)
}

func TestWithReconnectionZeroRetriesDisables(t *testing.T) {
	o := defaultOptions()
	WithReconnection(0, time.Second)(o)

	assert.Nil(t, o.recovery)
}

func TestWithoutReconnection(t *testing.T) {
	o := defaultOptions()
	WithoutReconnection()(o)

	assert.Nil(t, o.recovery)
}

func TestNewUnreachableBrokerWithOptions(t *testing.T) {
	client, err := New("amqp://guest:guest@127.0.0.1:1/", "dm-go-test", WithReconnection(2, time.Millisecond))

	require.Error(t, err)
	require.NotNil(t, client)
}

func TestPublishNilConnection(t *testing.T) {
	client := &Client{}

	err := client.Publish(context.Background(), "exchange", "key", Publishing{Body: []byte("payload")})

	assert.ErrorIs(t, err, amqp.ErrClosed)
}

func TestSubscribeNilConnection(t *testing.T) {
	client := &Client{}

	err := client.Subscribe(ConsumerConfig{}, func(ctx context.Context, msg *Message) error { return nil })

	assert.ErrorIs(t, err, amqp.ErrClosed)
}

func TestPingNilConnection(t *testing.T) {
	client := &Client{}

	assert.ErrorIs(t, client.Ping(), amqp.ErrClosed)
}

func TestPublishingAlias(t *testing.T) {
	var p Publishing = amqp.Publishing{ContentType: "application/json", Body: []byte(`{}`)}

	assert.Equal(t, "application/json", p.ContentType)
}

func TestLogStateChangesStopsWhenChannelCloses(t *testing.T) {
	states := make(chan *amqp.StateChanged, 4)
	states <- &amqp.StateChanged{From: amqp.StateOpen, To: amqp.StateReconnecting}
	states <- &amqp.StateChanged{From: amqp.StateReconnecting, To: amqp.StateOpen}
	states <- &amqp.StateChanged{From: amqp.StateReconnecting, To: amqp.StateClosed, Err: errors.New("retries exhausted")}
	close(states)

	done := make(chan struct{})
	go func() {
		logStateChanges("dm-go-test", states)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("logStateChanges did not return after the state channel was closed")
	}
}

// fakeAcknowledger records the ack/nack calls made through amqp.Delivery.
type fakeAcknowledger struct {
	mu      sync.Mutex
	acks    []uint64
	nacks   []uint64
	requeue []bool
	err     error
}

func (f *fakeAcknowledger) Ack(tag uint64, multiple bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acks = append(f.acks, tag)
	return f.err
}

func (f *fakeAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nacks = append(f.nacks, tag)
	f.requeue = append(f.requeue, requeue)
	return f.err
}

func (f *fakeAcknowledger) Reject(tag uint64, requeue bool) error {
	return f.Nack(tag, false, requeue)
}

func TestRequeueWrapsCause(t *testing.T) {
	cause := errors.New("db down")

	err := Requeue(cause)

	assert.ErrorIs(t, err, ErrRequeue)
	assert.ErrorIs(t, err, cause)
	assert.ErrorIs(t, Requeue(nil), ErrRequeue)
}

func TestHandleDeliveryAutoAckNeverAcks(t *testing.T) {
	ack := &fakeAcknowledger{}

	handleDelivery(amqp.Delivery{Acknowledger: ack, DeliveryTag: 1}, consumer{
		handler: func(ctx context.Context, msg *Message) error { return errors.New("boom") },
	})

	assert.Empty(t, ack.acks)
	assert.Empty(t, ack.nacks)
}

func TestHandleDeliveryManualAckOnSuccess(t *testing.T) {
	ack := &fakeAcknowledger{}

	handleDelivery(amqp.Delivery{Acknowledger: ack, DeliveryTag: 7}, consumer{
		manualAck: true,
		handler:   func(ctx context.Context, msg *Message) error { return nil },
	})

	assert.Equal(t, []uint64{7}, ack.acks)
	assert.Empty(t, ack.nacks)
}

func TestHandleDeliveryManualAckRequeuesOnErrRequeue(t *testing.T) {
	ack := &fakeAcknowledger{}

	handleDelivery(amqp.Delivery{Acknowledger: ack, DeliveryTag: 8}, consumer{
		manualAck: true,
		handler:   func(ctx context.Context, msg *Message) error { return Requeue(errors.New("try later")) },
	})

	assert.Empty(t, ack.acks)
	assert.Equal(t, []uint64{8}, ack.nacks)
	assert.Equal(t, []bool{true}, ack.requeue)
}

func TestHandleDeliveryManualAckRejectsOnOtherError(t *testing.T) {
	ack := &fakeAcknowledger{}

	handleDelivery(amqp.Delivery{Acknowledger: ack, DeliveryTag: 9}, consumer{
		manualAck: true,
		handler:   func(ctx context.Context, msg *Message) error { return errors.New("invalid payload") },
	})

	assert.Empty(t, ack.acks)
	assert.Equal(t, []uint64{9}, ack.nacks)
	assert.Equal(t, []bool{false}, ack.requeue)
}

func TestHandleDeliveryManualAckSurvivesAckError(t *testing.T) {
	ack := &fakeAcknowledger{err: amqp.ErrClosed}

	assert.NotPanics(t, func() {
		handleDelivery(amqp.Delivery{Acknowledger: ack, DeliveryTag: 1}, consumer{
			manualAck: true,
			handler:   func(ctx context.Context, msg *Message) error { return nil },
		})
	})
}

func TestHandleDeliveryTimeouts(t *testing.T) {
	deadlineOf := func(timeout time.Duration) (time.Time, bool) {
		var deadline time.Time
		var ok bool
		handleDelivery(amqp.Delivery{}, consumer{timeout: timeout, handler: func(ctx context.Context, msg *Message) error {
			deadline, ok = ctx.Deadline()
			return nil
		}})
		return deadline, ok
	}

	d, ok := deadlineOf(0)
	require.True(t, ok, "zero timeout must apply the default")
	assert.WithinDuration(t, time.Now().Add(DefaultHandlerTimeout), d, 2*time.Second)

	d, ok = deadlineOf(time.Minute)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(time.Minute), d, 2*time.Second)

	_, ok = deadlineOf(-1)
	assert.False(t, ok, "negative timeout must disable the deadline")
}

func TestChainOrdersClientThenConsumerMiddlewares(t *testing.T) {
	var order []string
	mw := func(name string) Middleware {
		return func(next SubscribeHandler) SubscribeHandler {
			return func(ctx context.Context, msg *Message) error {
				order = append(order, name+":before")
				err := next(ctx, msg)
				order = append(order, name+":after")
				return err
			}
		}
	}
	handler := func(ctx context.Context, msg *Message) error {
		order = append(order, "handler")
		return nil
	}

	h := chain(handler, []Middleware{mw("client1"), mw("client2")}, []Middleware{mw("consumer")})
	require.NoError(t, h(context.Background(), &Message{}))

	assert.Equal(t, []string{
		"client1:before", "client2:before", "consumer:before",
		"handler",
		"consumer:after", "client2:after", "client1:after",
	}, order)
}

func TestChainWithoutMiddlewaresReturnsHandler(t *testing.T) {
	called := false
	h := chain(func(ctx context.Context, msg *Message) error { called = true; return nil })

	require.NoError(t, h(context.Background(), &Message{}))
	assert.True(t, called)
}

func TestMiddlewareCanReplaceContextAndSeeError(t *testing.T) {
	type key struct{}
	var seen error
	mw := func(next SubscribeHandler) SubscribeHandler {
		return func(ctx context.Context, msg *Message) error {
			seen = next(context.WithValue(ctx, key{}, "traced"), msg)
			return seen
		}
	}
	var got any
	h := chain(func(ctx context.Context, msg *Message) error {
		got = ctx.Value(key{})
		return Requeue(errors.New("later"))
	}, []Middleware{mw})

	err := h(context.Background(), &Message{})

	assert.Equal(t, "traced", got)
	assert.ErrorIs(t, err, ErrRequeue)
	assert.ErrorIs(t, seen, ErrRequeue)
}

func TestWithMiddlewaresOption(t *testing.T) {
	o := defaultOptions()
	mw := func(next SubscribeHandler) SubscribeHandler { return next }
	WithMiddlewares(mw, mw)(o)

	assert.Len(t, o.middlewares, 2)
}

func TestDurableDefaultsToTrue(t *testing.T) {
	assert.True(t, durable(nil))
	assert.True(t, durable(boolPtr(true)))
	assert.False(t, durable(NotDurable))
	assert.False(t, durable(ExchangeOptions{Durable: NotDurable}.Durable))
	assert.True(t, durable(QueueOptions{}.Durable))
}
