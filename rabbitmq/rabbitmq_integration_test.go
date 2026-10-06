package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

// managementURL returns the RabbitMQ management API URL (with credentials) for tests that
// need to drop connections server-side, skipping the test when it is not set.
//
//	RABBITMQ_URI=amqp://guest:guest@localhost:5672/ RABBITMQ_MANAGEMENT_URL=http://guest:guest@localhost:15672 go test ./rabbitmq/ -run Integration
func managementURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("RABBITMQ_MANAGEMENT_URL")
	if u == "" {
		t.Skip("RABBITMQ_MANAGEMENT_URL not set; skipping RabbitMQ reconnection integration test")
	}
	return u
}

func TestIntegrationPublishIsConsumedBySubscriber(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-publish-exchange",
		ExchangeType: "topic",
		QueueName:    "dm-go-test-publish-queue",
		BindingKey:   "dm-go.publish",
		ConsumerName: "dm-go-test-publish-consumer",
	}
	received := make(chan *Message, 1)
	err = client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		received <- msg
		return nil
	})
	require.NoError(t, err)
	defer cleanupTopology(t, uri, cfg)

	body := []byte("published through dm-go " + time.Now().String())
	err = client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{
		ContentType: "text/plain",
		MessageId:   "dm-go-publish-1",
		Body:        body,
	})
	require.NoError(t, err)

	select {
	case msg := <-received:
		assert.Equal(t, body, msg.Body)
		assert.Equal(t, "dm-go-publish-1", msg.Delivery.MessageId)
		assert.Equal(t, "text/plain", msg.Delivery.ContentType)
		assert.Equal(t, cfg.BindingKey, msg.Delivery.RoutingKey)
	case <-time.After(5 * time.Second):
		t.Fatal("published message was not delivered to the subscribe handler")
	}
}

func TestIntegrationPublishReusesChannel(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-reuse-exchange",
		ExchangeType: "direct",
		QueueName:    "dm-go-test-reuse-queue",
		BindingKey:   "dm-go.reuse",
		ConsumerName: "dm-go-test-reuse-consumer",
	}
	received := make(chan *Message, 10)
	require.NoError(t, client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		received <- msg
		return nil
	}))
	defer cleanupTopology(t, uri, cfg)

	for i := 0; i < 5; i++ {
		require.NoError(t, client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{Body: []byte("n")}))
	}

	c := client.(*Client)
	c.mu.Lock()
	ch := c.publishCh
	c.mu.Unlock()
	require.NotNil(t, ch)
	assert.False(t, ch.IsClosed())

	for i := 0; i < 5; i++ {
		select {
		case <-received:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of 5 messages were delivered", i)
		}
	}
}

func TestIntegrationPublishAfterClose(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	client.Close()

	err = client.Publish(context.Background(), "dm-go-test-exchange", "dm-go.test", Publishing{Body: []byte("late")})

	assert.ErrorIs(t, err, amqp.ErrClosed)
}

func TestIntegrationReconnectRecoversPublishAndConsume(t *testing.T) {
	uri := brokerURI(t)
	mgmt := managementURL(t)

	connName := fmt.Sprintf("dm-go-reconnect-test-%d", time.Now().UnixNano())
	client, err := New(uri, connName, WithReconnection(10, time.Second))
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-reconnect-exchange",
		ExchangeType: "topic",
		QueueName:    "dm-go-test-reconnect-queue",
		BindingKey:   "dm-go.reconnect",
		ConsumerName: "dm-go-test-reconnect-consumer",
	}
	received := make(chan *Message, 100)
	require.NoError(t, client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		received <- msg
		return nil
	}))
	defer cleanupTopology(t, uri, cfg)

	require.NoError(t, client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{Body: []byte("before")}))
	select {
	case msg := <-received:
		assert.Equal(t, "before", string(msg.Body))
	case <-time.After(5 * time.Second):
		t.Fatal("message was not delivered before the connection drop")
	}

	dropConnection(t, mgmt, connName)

	// Publish may fail while the client is reconnecting, so keep trying until a message
	// published through the recovered connection reaches the recovered consumer.
	deadline := time.Now().Add(30 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("client did not recover publish and consume after the connection drop")
		}
		err := client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{Body: []byte("after")})
		if err == nil {
			select {
			case msg := <-received:
				assert.Equal(t, "after", string(msg.Body))
				assert.NoError(t, client.Ping())
				return
			case <-time.After(time.Second):
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// cleanupTopology deletes the queue and exchange created by a test using a dedicated connection.
func cleanupTopology(t *testing.T, uri string, cfg ConsumerConfig) {
	t.Helper()
	conn, err := amqp.Dial(uri)
	if err != nil {
		t.Logf("cleanup: dial failed: %v", err)
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Logf("cleanup: channel failed: %v", err)
		return
	}
	defer ch.Close()
	if _, err := ch.QueueDelete(cfg.QueueName, false, false, false); err != nil {
		t.Logf("cleanup: queue delete failed: %v", err)
	}
	if err := ch.ExchangeDelete(cfg.ExchangeName, false, false); err != nil {
		t.Logf("cleanup: exchange delete failed: %v", err)
	}
}

// dropConnection closes, through the management API, every broker connection whose client
// connection_name matches connName. The management API only lists a connection a few seconds
// after it is opened, so it polls until the connection shows up.
func dropConnection(t *testing.T, mgmt, connName string) {
	t.Helper()

	base, err := url.Parse(mgmt)
	require.NoError(t, err)
	user := base.User
	base.User = nil
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}

	do := func(method, path string) (*http.Response, error) {
		req, err := http.NewRequest(method, base.String()+path, nil)
		if err != nil {
			return nil, err
		}
		if user != nil {
			pass, _ := user.Password()
			req.SetBasicAuth(user.Username(), pass)
		}
		return httpClient.Do(req)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		require.False(t, time.Now().After(deadline), "connection %s never showed up in the management API", connName)

		resp, err := do(http.MethodGet, "/api/connections")
		require.NoError(t, err)
		var conns []struct {
			Name             string         `json:"name"`
			ClientProperties map[string]any `json:"client_properties"`
		}
		err = json.NewDecoder(resp.Body).Decode(&conns)
		resp.Body.Close()
		require.NoError(t, err)

		dropped := 0
		for _, c := range conns {
			if c.ClientProperties["connection_name"] != connName {
				continue
			}
			resp, err := do(http.MethodDelete, "/api/connections/"+url.PathEscape(c.Name))
			require.NoError(t, err)
			resp.Body.Close()
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			dropped++
		}
		if dropped > 0 {
			t.Logf("dropped %d connection(s) named %s", dropped, connName)
			return
		}
		time.Sleep(time.Second)
	}
}

func TestIntegrationManualAckRequeuesThenAcks(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-manualack-exchange",
		ExchangeType: "direct",
		QueueName:    "dm-go-test-manualack-queue",
		BindingKey:   "dm-go.manualack",
		ConsumerName: "dm-go-test-manualack-consumer",
		Consume:      ConsumeOptions{ManualAck: true, PrefetchCount: 1},
	}
	type seen struct {
		body        string
		redelivered bool
	}
	deliveries := make(chan seen, 10)
	var attempts int
	err = client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		attempts++
		deliveries <- seen{body: string(msg.Body), redelivered: msg.Delivery.Redelivered}
		if attempts == 1 {
			return Requeue(errors.New("first attempt fails"))
		}
		return nil
	})
	require.NoError(t, err)
	defer cleanupTopology(t, uri, cfg)

	require.NoError(t, client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{Body: []byte("retry-me")}))

	first := waitFor(t, deliveries, "first delivery")
	assert.Equal(t, "retry-me", first.body)
	assert.False(t, first.redelivered)

	second := waitFor(t, deliveries, "redelivery after requeue")
	assert.Equal(t, "retry-me", second.body)
	assert.True(t, second.redelivered, "the requeued message must come back flagged as redelivered")

	select {
	case extra := <-deliveries:
		t.Fatalf("message was delivered a third time after being acked: %+v", extra)
	case <-time.After(time.Second):
	}
	assert.Equal(t, 0, queueMessageCount(t, uri, cfg.QueueName), "acked message must leave the queue")
}

func TestIntegrationManualAckRejectDropsMessage(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-reject-exchange",
		ExchangeType: "direct",
		QueueName:    "dm-go-test-reject-queue",
		BindingKey:   "dm-go.reject",
		ConsumerName: "dm-go-test-reject-consumer",
		Consume:      ConsumeOptions{ManualAck: true},
	}
	deliveries := make(chan string, 10)
	err = client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		deliveries <- string(msg.Body)
		return errors.New("invalid payload")
	})
	require.NoError(t, err)
	defer cleanupTopology(t, uri, cfg)

	require.NoError(t, client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{Body: []byte("bad")}))

	assert.Equal(t, "bad", waitFor(t, deliveries, "first delivery"))
	select {
	case body := <-deliveries:
		t.Fatalf("rejected message %q was delivered again", body)
	case <-time.After(time.Second):
	}
	assert.Equal(t, 0, queueMessageCount(t, uri, cfg.QueueName), "rejected message must not stay in the queue")
}

func TestIntegrationMiddlewaresRunAroundHandler(t *testing.T) {
	uri := brokerURI(t)

	var order []string
	mw := func(name string) Middleware {
		return func(next SubscribeHandler) SubscribeHandler {
			return func(ctx context.Context, msg *Message) error {
				order = append(order, name)
				return next(ctx, msg)
			}
		}
	}
	client, err := New(uri, "dm-go-integration-test", WithMiddlewares(mw("client")))
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-middleware-exchange",
		ExchangeType: "direct",
		QueueName:    "dm-go-test-middleware-queue",
		BindingKey:   "dm-go.middleware",
		ConsumerName: "dm-go-test-middleware-consumer",
		Middlewares:  []Middleware{mw("consumer")},
		Queue:        QueueOptions{Args: Table{"x-queue-type": "classic"}},
	}
	done := make(chan struct{}, 1)
	err = client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		order = append(order, "handler")
		done <- struct{}{}
		return nil
	})
	require.NoError(t, err)
	defer cleanupTopology(t, uri, cfg)

	require.NoError(t, client.Publish(context.Background(), cfg.ExchangeName, cfg.BindingKey, Publishing{Body: []byte("x")}))
	waitFor(t, done, "handler")

	assert.Equal(t, []string{"client", "consumer", "handler"}, order)
}

func waitFor[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

// queueMessageCount returns the number of ready messages in the queue, using a passive declare.
func queueMessageCount(t *testing.T, uri, queue string) int {
	t.Helper()
	conn, err := amqp.Dial(uri)
	require.NoError(t, err)
	defer conn.Close()
	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()
	q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	require.NoError(t, err)
	return q.Messages
}

func TestIntegrationNotDurableDeclaresTransientTopology(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-transient-exchange",
		ExchangeType: "direct",
		QueueName:    "dm-go-test-transient-queue",
		BindingKey:   "dm-go.transient",
		ConsumerName: "dm-go-test-transient-consumer",
		Exchange:     ExchangeOptions{Durable: NotDurable},
		Queue:        QueueOptions{Durable: NotDurable},
	}
	require.NoError(t, client.Subscribe(cfg, func(ctx context.Context, msg *Message) error { return nil }))
	defer cleanupTopology(t, uri, cfg)

	// Re-declaring with durable=true must be refused by the broker (PRECONDITION_FAILED, inequivalent
	// arg 'durable'), which proves the topology was declared transient. A passive declare would not
	// do: the broker only checks existence on passive declares and ignores the flags.
	conn, err := amqp.Dial(uri)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	require.Error(t, err, "queue must have been declared transient")
	assert.Contains(t, err.Error(), "PRECONDITION_FAILED")

	ch, err = conn.Channel()
	require.NoError(t, err)
	err = ch.ExchangeDeclare(cfg.ExchangeName, cfg.ExchangeType, true, false, false, false, nil)
	require.Error(t, err, "exchange must have been declared transient")
	assert.Contains(t, err.Error(), "PRECONDITION_FAILED")

	ch, err = conn.Channel()
	require.NoError(t, err)
	defer ch.Close()
	_, err = ch.QueueDeclare(cfg.QueueName, false, false, false, false, nil)
	assert.NoError(t, err, "re-declare with durable=false must match the transient queue")
	err = ch.ExchangeDeclare(cfg.ExchangeName, cfg.ExchangeType, false, false, false, false, nil)
	assert.NoError(t, err, "re-declare with durable=false must match the transient exchange")
}

func TestIntegrationDefaultTopologyIsDurable(t *testing.T) {
	uri := brokerURI(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-durable-exchange",
		ExchangeType: "direct",
		QueueName:    "dm-go-test-durable-queue",
		BindingKey:   "dm-go.durable",
		ConsumerName: "dm-go-test-durable-consumer",
	}
	require.NoError(t, client.Subscribe(cfg, func(ctx context.Context, msg *Message) error { return nil }))
	defer cleanupTopology(t, uri, cfg)

	conn, err := amqp.Dial(uri)
	require.NoError(t, err)
	defer conn.Close()
	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()
	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	assert.NoError(t, err, "default queue must be durable")
	err = ch.ExchangeDeclare(cfg.ExchangeName, cfg.ExchangeType, true, false, false, false, nil)
	assert.NoError(t, err, "default exchange must be durable")

	ch, err = conn.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(cfg.QueueName, false, false, false, false, nil)
	require.Error(t, err, "re-declare with durable=false must be refused for the durable queue")
	assert.Contains(t, err.Error(), "PRECONDITION_FAILED")
}
