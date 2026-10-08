package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// TestIntegrationTraceIsPropagatedFromPublishToConsume is the end-to-end check of the AMQP
// instrumentation: a message published inside a span, through a real broker, is processed by a
// consumer span of the same trace whose parent is the producer span.
func TestIntegrationTraceIsPropagatedFromPublishToConsume(t *testing.T) {
	uri := brokerURI(t)
	exp := installTestTracer(t)

	client, err := New(uri, "dm-go-integration-test")
	require.NoError(t, err)
	defer client.Close()

	cfg := ConsumerConfig{
		ExchangeName: "dm-go-test-trace-exchange",
		ExchangeType: "topic",
		QueueName:    "dm-go-test-trace-queue",
		BindingKey:   "dm-go.trace",
		ConsumerName: "dm-go-test-trace-consumer",
	}
	type seen struct {
		msg *Message
		sc  trace.SpanContext
	}
	received := make(chan seen, 1)
	err = client.Subscribe(cfg, func(ctx context.Context, msg *Message) error {
		received <- seen{msg: msg, sc: trace.SpanFromContext(ctx).SpanContext()}
		return nil
	})
	require.NoError(t, err)
	defer cleanupTopology(t, uri, cfg)

	rootCtx, root := tracer().Start(context.Background(), "test-root")
	err = client.Publish(rootCtx, cfg.ExchangeName, cfg.BindingKey, Publishing{
		MessageId: "dm-go-trace-1",
		Headers:   Table{"x-custom": "keep-me"},
		Body:      []byte("traced"),
	})
	require.NoError(t, err)
	root.End()

	got := waitFor(t, received, "traced message")
	assert.Equal(t, "keep-me", got.msg.Delivery.Headers["x-custom"], "custom headers travel with the trace context")
	assert.Contains(t, got.msg.Delivery.Headers, "traceparent")

	// The consumer span ends after the handler returns, give the syncer a moment to export it.
	var spans = exp.GetSpans()
	for deadline := time.Now().Add(2 * time.Second); len(spans) < 3 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
		spans = exp.GetSpans()
	}

	rootStub := spanNamed(t, spans, "test-root")
	publish := spanNamed(t, spans, "publish "+cfg.ExchangeName)
	process := spanNamed(t, spans, "process "+cfg.QueueName)

	traceID := rootStub.SpanContext.TraceID()
	assert.Equal(t, traceID, publish.SpanContext.TraceID(), "producer span belongs to the caller's trace")
	assert.Equal(t, traceID, process.SpanContext.TraceID(), "consumer span belongs to the same trace")
	assert.Equal(t, traceID, got.sc.TraceID(), "handler context carries the same trace")

	assert.Equal(t, rootStub.SpanContext.SpanID(), publish.Parent.SpanID(), "producer span is a child of the caller span")
	assert.Equal(t, publish.SpanContext.SpanID(), process.Parent.SpanID(), "consumer span is a child of the producer span")
	assert.True(t, process.Parent.IsRemote(), "the producer context arrived through the broker")
	assert.Equal(t, trace.SpanKindProducer, publish.SpanKind)
	assert.Equal(t, trace.SpanKindConsumer, process.SpanKind)
}
