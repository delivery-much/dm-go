package rabbitmq

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// installTestTracer registers an in-memory tracer provider and the W3C propagator as the OTel
// globals for the duration of the test, returning the exporter that collects the ended spans.
func installTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		installNoopTracer(t)
	})
	return exp
}

// installNoopTracer resets the OTel globals to the behaviour of a service without telemetry.
func installNoopTracer(t *testing.T) {
	t.Helper()
	otel.SetTracerProvider(noop.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
}

func spanNamed(t *testing.T, spans tracetest.SpanStubs, name string) tracetest.SpanStub {
	t.Helper()
	for _, s := range spans {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("span %q not found in %d exported spans", name, len(spans))
	return tracetest.SpanStub{}
}

func attrValue(attrs []attribute.KeyValue, key attribute.Key) attribute.Value {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value
		}
	}
	return attribute.Value{}
}

func TestHeaderCarrier(t *testing.T) {
	c := headerCarrier(Table{"traceparent": "00-abc", "x-count": int32(3)})

	assert.Equal(t, "00-abc", c.Get("traceparent"))
	assert.Equal(t, "", c.Get("x-count"), "non-string headers are not propagation fields")
	assert.Equal(t, "", c.Get("missing"))

	c.Set("tracestate", "vendor=1")
	assert.Equal(t, "vendor=1", c["tracestate"])
	assert.ElementsMatch(t, []string{"traceparent", "x-count", "tracestate"}, c.Keys())
}

func TestStartPublishSpanInjectsTraceContextWithoutMutatingCallerHeaders(t *testing.T) {
	exp := installTestTracer(t)

	parentCtx, parent := tracer().Start(context.Background(), "test-parent")
	original := Table{"x-custom": "keep-me"}
	msg := Publishing{MessageId: "msg-1", Body: []byte("hello"), Headers: original}

	_, span := startPublishSpan(parentCtx, "orders", "orders.created", &msg)
	span.End()
	parent.End()

	traceparent, ok := msg.Headers["traceparent"].(string)
	require.True(t, ok, "traceparent header must be injected")
	assert.Contains(t, traceparent, span.SpanContext().TraceID().String())
	assert.Contains(t, traceparent, span.SpanContext().SpanID().String())
	assert.Equal(t, "keep-me", msg.Headers["x-custom"], "existing headers are preserved")
	assert.Equal(t, Table{"x-custom": "keep-me"}, original, "caller's table is not mutated")

	stub := spanNamed(t, exp.GetSpans(), "publish orders")
	assert.Equal(t, trace.SpanKindProducer, stub.SpanKind)
	assert.Equal(t, parent.SpanContext().SpanID(), stub.Parent.SpanID())
	assert.Equal(t, "rabbitmq", attrValue(stub.Attributes, semconv.MessagingSystemKey).AsString())
	assert.Equal(t, "publish", attrValue(stub.Attributes, semconv.MessagingOperationTypeKey).AsString())
	assert.Equal(t, "publish", attrValue(stub.Attributes, semconv.MessagingOperationNameKey).AsString())
	assert.Equal(t, "orders", attrValue(stub.Attributes, semconv.MessagingDestinationNameKey).AsString())
	assert.Equal(t, "orders.created", attrValue(stub.Attributes, semconv.MessagingRabbitmqDestinationRoutingKeyKey).AsString())
	assert.Equal(t, "msg-1", attrValue(stub.Attributes, semconv.MessagingMessageIDKey).AsString())
	assert.Equal(t, int64(5), attrValue(stub.Attributes, semconv.MessagingMessageBodySizeKey).AsInt64())
}

func TestStartPublishSpanWithoutProviderLeavesHeadersUntouched(t *testing.T) {
	installNoopTracer(t)

	msg := Publishing{Body: []byte("hello")}
	_, span := startPublishSpan(context.Background(), "orders", "orders.created", &msg)
	span.End()

	assert.Nil(t, msg.Headers, "no provider: nothing is injected and nil headers stay nil")
}

func TestHandleDeliveryContinuesTraceFromHeaders(t *testing.T) {
	exp := installTestTracer(t)

	// Simulate a producer on another service: a remote span context serialised as W3C traceparent.
	remote := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36},
		SpanID:     trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
		TraceFlags: trace.FlagsSampled,
	})
	headers := Table{}
	otel.GetTextMapPropagator().Inject(trace.ContextWithRemoteSpanContext(context.Background(), remote), headerCarrier(headers))
	require.Contains(t, headers, "traceparent")

	var handlerSC trace.SpanContext
	handleDelivery(amqp.Delivery{
		Headers:     headers,
		Exchange:    "orders",
		RoutingKey:  "orders.created",
		MessageId:   "msg-7",
		DeliveryTag: 7,
		Body:        []byte("hi"),
	}, consumer{
		queue: "orders.created.q",
		handler: func(ctx context.Context, msg *Message) error {
			handlerSC = trace.SpanFromContext(ctx).SpanContext()
			return nil
		},
	})

	require.True(t, handlerSC.IsValid(), "handler context must carry the consumer span")
	assert.Equal(t, remote.TraceID(), handlerSC.TraceID(), "consumer span joins the producer trace")

	stub := spanNamed(t, exp.GetSpans(), "process orders.created.q")
	assert.Equal(t, trace.SpanKindConsumer, stub.SpanKind)
	assert.Equal(t, remote.SpanID(), stub.Parent.SpanID(), "consumer span is a child of the producer span")
	assert.True(t, stub.Parent.IsRemote())
	assert.Equal(t, codes.Unset, stub.Status.Code)
	assert.Equal(t, "process", attrValue(stub.Attributes, semconv.MessagingOperationTypeKey).AsString())
	assert.Equal(t, "process", attrValue(stub.Attributes, semconv.MessagingOperationNameKey).AsString())
	assert.Equal(t, "orders.created.q", attrValue(stub.Attributes, semconv.MessagingDestinationNameKey).AsString())
	assert.Equal(t, "orders", attrValue(stub.Attributes, messagingDestinationPublishNameKey).AsString())
	assert.Equal(t, "orders.created", attrValue(stub.Attributes, semconv.MessagingRabbitmqDestinationRoutingKeyKey).AsString())
	assert.Equal(t, "msg-7", attrValue(stub.Attributes, semconv.MessagingMessageIDKey).AsString())
	assert.Equal(t, int64(7), attrValue(stub.Attributes, semconv.MessagingRabbitmqMessageDeliveryTagKey).AsInt64())
}

func TestHandleDeliveryWithoutHeadersStartsNewTrace(t *testing.T) {
	exp := installTestTracer(t)

	handleDelivery(amqp.Delivery{RoutingKey: "legacy"}, consumer{
		queue:   "legacy.q",
		handler: func(ctx context.Context, msg *Message) error { return nil },
	})

	stub := spanNamed(t, exp.GetSpans(), "process legacy.q")
	assert.True(t, stub.SpanContext.IsValid())
	assert.False(t, stub.Parent.IsValid(), "a message without trace headers starts a root span")
}

func TestHandleDeliveryRecordsHandlerError(t *testing.T) {
	exp := installTestTracer(t)

	boom := errors.New("boom")
	handleDelivery(amqp.Delivery{}, consumer{
		queue:   "errors.q",
		handler: func(ctx context.Context, msg *Message) error { return boom },
	})

	stub := spanNamed(t, exp.GetSpans(), "process errors.q")
	assert.Equal(t, codes.Error, stub.Status.Code)
	assert.Equal(t, "boom", stub.Status.Description)
	require.Len(t, stub.Events, 1)
	assert.Equal(t, "exception", stub.Events[0].Name)
}

func TestMiddlewaresSeeTheConsumerSpan(t *testing.T) {
	installTestTracer(t)

	var mwSC, handlerSC trace.SpanContext
	mw := func(next SubscribeHandler) SubscribeHandler {
		return func(ctx context.Context, msg *Message) error {
			mwSC = trace.SpanFromContext(ctx).SpanContext()
			return next(ctx, msg)
		}
	}
	handler := func(ctx context.Context, msg *Message) error {
		handlerSC = trace.SpanFromContext(ctx).SpanContext()
		return nil
	}
	handleDelivery(amqp.Delivery{}, consumer{queue: "mw.q", handler: chain(handler, []Middleware{mw})})

	require.True(t, mwSC.IsValid(), "middlewares run inside the consumer span")
	assert.Equal(t, mwSC, handlerSC)
}
