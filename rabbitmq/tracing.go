package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation scope of the spans created by this package.
const tracerName = "github.com/delivery-much/dm-go/rabbitmq"

// messagingDestinationPublishNameKey is the exchange the message was originally published to.
// semconv v1.27.0 dropped the "messaging.destination_publish.name" attribute without a replacement,
// so the key is kept here to preserve the attribute on consumer spans.
const messagingDestinationPublishNameKey = attribute.Key("messaging.destination_publish.name")

// tracer resolves the tracer lazily, so it follows the provider registered by telemetry.InitWithConfig
// even when the client is created before it. Without a provider the OTel API is a no-op.
func tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// headerCarrier adapts the AMQP message headers to the OpenTelemetry propagation carrier,
// so the trace context travels with the message (traceparent, tracestate, baggage).
type headerCarrier Table

func (c headerCarrier) Get(key string) string {
	if v, ok := c[key].(string); ok {
		return v
	}
	return ""
}

func (c headerCarrier) Set(key, value string) {
	c[key] = value
}

func (c headerCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// startPublishSpan starts the producer span of a publish and injects its context into the message
// headers. The headers of msg are copied before the injection, so the caller's Table is never mutated.
// Injection depends on the global propagator and on the trace context in ctx, not on the tracer
// provider. When there is nothing to inject (no-op propagator or no trace context) msg.Headers is
// left as given.
func startPublishSpan(ctx context.Context, exchange, routingKey string, msg *Publishing) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemRabbitmq,
		semconv.MessagingOperationTypePublish,
		semconv.MessagingOperationName("publish"),
		semconv.MessagingDestinationName(exchange),
		semconv.MessagingRabbitmqDestinationRoutingKey(routingKey),
		semconv.MessagingMessageBodySize(len(msg.Body)),
	}
	if msg.MessageId != "" {
		attrs = append(attrs, semconv.MessagingMessageID(msg.MessageId))
	}

	ctx, span := tracer().Start(ctx, "publish "+exchange,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attrs...),
	)

	headers := make(Table, len(msg.Headers)+3)
	for k, v := range msg.Headers {
		headers[k] = v
	}
	otel.GetTextMapPropagator().Inject(ctx, headerCarrier(headers))
	if len(headers) > 0 {
		msg.Headers = headers
	}
	return ctx, span
}

// startConsumeSpan extracts the trace context from the delivery headers and starts the consumer span
// as a child of the producer span, so publish and process are correlated in the same trace.
func startConsumeSpan(ctx context.Context, queue string, d amqp.Delivery) (context.Context, trace.Span) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, headerCarrier(d.Headers))

	attrs := []attribute.KeyValue{
		semconv.MessagingSystemRabbitmq,
		semconv.MessagingOperationTypeProcess,
		semconv.MessagingOperationName("process"),
		semconv.MessagingDestinationName(queue),
		semconv.MessagingRabbitmqDestinationRoutingKey(d.RoutingKey),
		semconv.MessagingRabbitmqMessageDeliveryTag(int(d.DeliveryTag)),
		semconv.MessagingMessageBodySize(len(d.Body)),
	}
	if d.Exchange != "" {
		attrs = append(attrs, messagingDestinationPublishNameKey.String(d.Exchange))
	}
	if d.MessageId != "" {
		attrs = append(attrs, semconv.MessagingMessageID(d.MessageId))
	}

	return tracer().Start(ctx, "process "+queue,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

// endSpan records err on the span, when there is one, and ends it.
func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}
