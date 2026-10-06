<p align="center"><img src="assets/gopher.png" width="350"></p>

<h1 align="center">
  dm-go
</h1>

* [Overview](#overview)
* [Packages](#packages)
	* [Logger](#logger)
	* [Middleware](#middleware)
	* [Render](#render)
	* [String Utils](#string-utils)
	* [Request](#request)
	* [Telemetry](#telemetry)
	* [RabbitMQ](#rabbitmq)

## Overview

Reusable packages and frameworks for Go services.

## Installation

```bash
go get github.com/delivery-much/dm-go
```

## Packages

Implemented packages in project, their use and some examples.

### Logger

Package for log application, ready for production (JSON) and development. Construct the log with some parameters: if is json, the level and some base fields that will appear in all output logs. Using the Sugar [zap](https://github.com/uber-go/zap) package.

Level: `debug`, `info` (default), `warn`, `error`, `fatal`

Example:

```go
ctx := context.TODO()
config := logger.Configuration{
    IsJSON: true,
    Level:  "info",
    BaseFields: logger.BaseFields{
        ServiceName: "default-service",
        CodeVersion: "1.0.0",
        Env:         "production",
    },
}
err := logger.NewLogger(config)
if err != nil {
    panic(err)
}

logger.Infow(ctx, "failed to fetch URL",
    // Key and value after the message
    "url", "www.google.com",
    "attempt", 3,
    "backoff", time.Second,
)
```

By default, the logger package needs a context to log extra information and fields.
But the user can use the logger without the need to provide a context, like such:

```go
ctx := context.TODO()
logger.Info(ctx, "Hello!")
logger.NoCTX().Info("Hello!")
```

The context variables that will be searched and used can be personalized
by the user in the `Configuration` passed to the `NewLogger` function, using the `CTXFields` value.

The `CTXFields` value maps the field that the logger should look for in the context,
to the field that it should use in the log when logging the correspondent value.

If the specified context does not have the key, it will ignore the field.

In addition, by default, the logger package will always use the middleware package
to look for a request id in the context, if it finds, the request id will be logged in the `request_id` field

#### OpenTelemetry log emission

Besides writing to stdout, the logger also emits every log record via OpenTelemetry when **both**
conditions hold:

1. The `OTEL_EXPORTER_OTLP_ENDPOINT` env var is set (this exact variable is the on/off switch;
   signal-specific variants don't count), and
2. a `LoggerProvider` is installed globally — done by calling the [Telemetry](#telemetry)
   package's `telemetry.Init(ctx)` **before** `logger.NewLogger`.

Emission can be opted out per service with `Configuration{DisableOpenTelemetry: true}`.

Compatibility: this package requires Go 1.27 and `go.opentelemetry.io/otel/log` 0.22.0 or newer.
The `telemetry` package lives in this same module, so both always ship against the same
`otel/log` version.

The `BaseFields` (service name, env, code version) appear only in the stdout logs. OTel log
records don't repeat them: the equivalent resource attributes (`service.name`,
`deployment.environment`, `service.version`) come from the `telemetry` package, which reads them
from the `SERVICE_NAME`, `ENVIRONMENT`, and `CODE_VERSION` env vars.

Ex.:
```go
    myCTXKey := "context-key"
    ctx = context.WithValue(context.TODO(), myCTXKey, "CTX VALUE!!")
	ctx := context.WithValue(ctx, middleware.RequestIDKey, "reqID")

    config := logger.Configuration{
        IsJSON: true,
        Level:  "info",
        CTXFields: map[any]string{
            myCTXKey: "log_field",
        }
    }

    // will log: {"message": "HELLO!!", "log_field": "CTX VALUE!!", "request_id": "reqID"}
    logger.Info(ctx, "HELLO!!")

    // will log: {"message": "HELLO!!"}
    logger.Info(context.TODO(), "HELLO!!")
```



### Middleware

Package with some middleware for routes and service.

Example:

```go
// Middleware for generate or inject request id in context of request.
router.Use(
    middleware.RequestID("Key-Request-Id"),
)
```

### Render

Package with some helpers for render responses. To respond in JSON format.

### String Utils

Package with some utility functions for string transformation

#### MaskString(str string) string
- MaskString masks the last half of a given string, changing any letter or number character to '*'
Example:
```golang
MaskString("examplestring") // returns "exampl*******"
MaskString("") // returns ""
```

#### MaskEmail(email string) string
- MaskEmail masks an email string,
leaving only the first four letters of the email id (i.e the part before the '@') and the email domain unmasked.
if the email id has 4 or less characters, leaves only 1 character unmasked.
Example:
```golang
MaskEmail("email_id@domain.com") // returns "emai*_**@domain.com"
MaskEmail("0101@domain.com") // returns "0***@domain.com"
MaskEmail("notanemail.com") // returns "notanemail.com"
MaskEmail("") //returns ""
```

### Request

Package that serves as an abstraction of the code used to perform HTTP requests.

With this package, it's possible to perform a request and easily handle the response, with resources for status validation and transformation of the response body.

In addition, there are abstractions that allow the user to perform `GET`, `POST`, `PUT`, `PATCH` and `DELETE` requests in a simpler way.

#### Perform a request

To perform a request, you can define some parameters:

- **method** [string]: the HTTP method from request (e.g.: `GET | POST | PUT | PATCH | DELETE`)
- **url** [url.URL]: a struct that represents an URL (from package `net/url`)
- **headers\*** [map[string]string]: a key-value map that represents the request headers
- **body\*** [io.Reader]: an interface that wraps the request body as a byte array (from package `io`)

> Note: params with a single asterisk (\*) are optional.

Example:

```golang
package main

import (
	"io"
	"net/url"
	"strings"

	"github.com/delivery-much/dm-go/request"
)

func main() {
	client := request.Client{}

	method := "POST"
	url := &url.URL{
		Scheme: "http",
		Host:   "localhost",
		Path:   "/users",
	}

	headers := map[string]string{
		"Accept": "application/json",
	}

	body := io.NopCloser(strings.NewReader(`{"name": "John Doe"}`))
	params := request.Params{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	// using the method Do
	res, err := client.Do(params)

	// using the method Get
	res, err = client.Get(url)
	res, err = client.Get(url, headers)


	// using the method Post
	res, err = client.Post(url, body)
	res, err = client.Post(url, body, headers)

	// using the method Put
	res, err = client.Put(url, body)
	res, err = client.Put(url, body, headers)

	// using the method Patch
	res, err = client.Patch(url, body)
	res, err = client.Patch(url, body, headers)

	// using the method Delete
	res, err = client.Delete(url)
	res, err = client.Delete(url, headers)
}

```

#### Dealing with response

To deal with the request response, you can use some resources provided by the library.

The library allows the user to check the response status and parse the response body easily. Example:

```golang
package main

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/delivery-much/dm-go/request"
)

func main() {
	client := request.Client{}

	method := "POST"
	url := &url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   "/users",
	}

	headers := map[string]string{
		"Accept": "application/json",
	}

	body := io.NopCloser(strings.NewReader(`{"name": "John Doe"}`))
	params := request.Params{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	// using the method Do
	res, err := client.Do(params)
	if err != nil {
		panic(err)
	}

	// Checks if the status code is from a successful response.
	// Successful responses have a status between 200 and 299.
	if res.IsSuccessCode() {
		fmt.Printf("Request succeeds with status %d\n", res.StatusCode)
	}

	// Checks if the status code is from a failure response.
	// Failure responses have a status different than a successful response.
	if res.IsFailureCode() {
		fmt.Printf("Request fails with status %d\n", res.StatusCode)
	}

	/*

	Presuming that response will return the status 201 and following body:

	{
	   "id": "66f467e3ad40102788ac1c93",
	   "name": "John Doe",
	}

	*/
	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var user User
	res.DecodeJSON(&user)

	fmt.Println(user)

    /*

    Expected output:

    Request succeeds with status 201
    {66f467e3ad40102788ac1c93 John Doe}

    */
}


```

> Note: To properly decode the response body into a defined object, it must be consistent with the expected response type.
> Example: if a response body in JSON format is expected, the `json` tags must be defined in the struct
> that will be mapped as the response body.

### Telemetry

Shared OpenTelemetry SDK setup for Go services. Bootstraps all three OTel signals (traces,
metrics, logs) with a single `Init()` call, exporting via OTLP HTTP to the local collector agent.
Design notes and decision documents live in [docs/](./docs/).

#### Quick Start

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"

    "github.com/delivery-much/dm-go/telemetry"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    shutdown, err := telemetry.Init(ctx)
    if err != nil {
        log.Fatalf("telemetry setup failed: %v", err)
    }
    defer shutdown(context.Background())

    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    mux.HandleFunc("GET /api/hello", helloHandler)

    handler := telemetry.Middleware("my-service")(mux)
    http.ListenAndServe(":3000", handler)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
    ctx, end := telemetry.StartSpan(r.Context(), "hello-logic")
    defer end()

    // add attributes to the span
    telemetry.SetAttributes(ctx, attribute.String("user.id", "123"))

    // errors are recorded on the span and set status to Error
    if err := doSomething(ctx); err != nil {
        telemetry.RecordError(ctx, err)
        http.Error(w, "error", 500)
        return
    }

    w.Write([]byte("hello"))
}
```

#### What's Included

| File | What it does |
|------|-------------|
| `telemetry/config.go` | Defaults loaded from environment (endpoint, sampler, signal toggles) |
| `telemetry/telemetry.go` | `Init()`, `Middleware()`, `Transport()`, `HTTPClient()` |
| `telemetry/exporters.go` | OTLP HTTP exporter builders for traces, metrics, logs |
| `telemetry/trace.go` | Helpers: `StartSpan()`, `RecordError()`, `TraceIDFromContext()` |
| `telemetry/meter.go` | Helpers: `Meter()`, `WithHTTPDurationBuckets()` for custom metrics |

#### Configuration

**Telemetry is active only when an OTLP endpoint is configured** — typically via the
`OTEL_EXPORTER_OTLP_ENDPOINT` env var (or a signal-specific variant, or `Config.OTLPEndpoint`).
Without one, `Init` installs no providers, logs a single "OpenTelemetry disabled" line, and
returns a no-op shutdown — the service runs with zero telemetry overhead.

**Convention: services configure telemetry via `OTEL_EXPORTER_OTLP_ENDPOINT`.** The other
endpoint sources (signal-specific env vars, `Config.OTLPEndpoint`) also activate this lib, but
they do **not** activate OTel log emission in `dm-go/logger`, which checks that exact variable —
`Init` prints a startup warning when it detects this mismatch.

`Init(ctx)` uses environment variables and library defaults. To configure programmatically,
use `InitWithConfig` — zero-value fields still fall back to env vars and defaults:

```go
shutdown, err := telemetry.InitWithConfig(ctx, telemetry.Config{
    ServiceName:  "my-service",
    SamplerRatio: telemetry.Float64(0.1), // nil = 1.0; 0.0 = never sample (parent-based)
    OTLPHeaders:  map[string]string{"x-api-key": key},
})
```

| Field | Default | Description |
|-------|---------|-------------|
| `SERVICE_NAME` | `""` | `service.name` resource attribute |
| `CODE_VERSION` | `""` | `service.version` |
| `ENVIRONMENT` | `""` | `deployment.environment` |
| `ServiceInstanceID` | `hostname:SERVICE_PORT` | `service.instance.id`, the per-replica identity behind the Prometheus `instance` label (see below) |
| `SERVICE_PORT` | `""` | Port appended to the derived `service.instance.id` |
| `OTLPEndpoint` | *(none — telemetry disabled)* | Collector endpoint |
| `OTLPInsecure` | `true` for scheme-less endpoints | HTTP instead of HTTPS; use an `https://` endpoint URL for TLS, or set `OTEL_EXPORTER_OTLP_INSECURE` |
| `SamplerRatio` | `1.0` | Trace sampling (0.0–1.0) |
| `MetricInterval` | `60s` | Metric export interval |
| `MetricCardinalityLimit` | `2000` (SDK default) | Max attribute sets per instrument per collect cycle; overflow is aggregated into `otel.metric.overflow=true`. `telemetry.Int(0)` disables the limit |
| `TracingEnabled` | `true` | Enable/disable traces |
| `MetricsEnabled` | `true` | Enable/disable metrics |
| `LoggingEnabled` | `true` | Enable/disable logs |

The SDK also reads OTel env vars (`SERVICE_NAME`, `CODE_VERSION`,
`ENVIRONMENT`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_RESOURCE_ATTRIBUTES`, etc.)
automatically.

##### Instance identity

Every process needs its own `service.instance.id`: the collector turns it into the Prometheus
`instance` label, and without it the counters of all replicas — and of consecutive deploys —
land on one series, so `rate()` and `increase()` report nonsense. The library derives
`hostname:port` when nothing else provides it (`SERVICE_PORT` or `Config.ServicePort` supplies
the port). That gives the pod name on Kubernetes, the host name on EC2 and the container id
under Docker; the port keeps two processes on one host apart and, unlike a PID or a random id,
does not change on restart. Precedence, highest first: `Config.ServiceInstanceID`, then
`service.instance.id` inside `OTEL_RESOURCE_ATTRIBUTES`, then the derived default.

Signal-specific endpoints use the standard names `OTEL_EXPORTER_OTLP_{TRACES,METRICS,LOGS}_ENDPOINT`.
The names `OTEL_{TRACES,METRICS,LOGS}_OTLP_ENDPOINT` are legacy internal aliases kept for
compatibility. `OTEL_METRICS_HTTP_ENDPOINT` is also accepted as a legacy metrics-only alias.
Legacy aliases apply **only when no base endpoint is configured** (neither `Config.OTLPEndpoint`
nor `OTEL_EXPORTER_OTLP_ENDPOINT`): once a collector is explicitly configured, a stale alias
left over from an older telemetry stack cannot silently redirect a single signal elsewhere —
use the standard signal-specific names to intentionally override per signal.
Prefer the standard names. (And remember: only `OTEL_EXPORTER_OTLP_ENDPOINT` activates
`dm-go/logger`.)

#### Local OpenTelemetry stack

This repository includes a Docker Compose stack (`docker-compose.yml` + `observability/`) for local telemetry tests with:

| Signal | Receiver | Storage/UI |
|--------|----------|------------|
| Traces | OpenTelemetry Collector | Tempo + Grafana |
| Metrics | OpenTelemetry Collector | Prometheus + Grafana |
| Logs | OpenTelemetry Collector | Loki + Grafana |

Start it with:

```sh
docker compose up -d
```

Configure a local Go service using `dm-go/telemetry` and `dm-go/logger` with:

```sh
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_INSECURE=true
export SERVICE_NAME=my-service
export SERVICE_PORT=3000
export ENVIRONMENT=local
export CODE_VERSION=dev
```

If the service also runs inside Docker Compose on the same network, use
`OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318` instead. Grafana is exposed at
<http://localhost:3001>, Prometheus at <http://localhost:9090>, Tempo at
<http://localhost:3200>, and Loki at <http://localhost:3100>.

#### HTTP Instrumentation

```go
// Server — wraps handler with automatic span + metrics.
// Requests to /health are not traced.
handler := telemetry.Middleware("my-service")(mux)

// Server spans are named "{METHOD} {pattern}" when the router sets r.Pattern
// (net/http ServeMux on Go 1.22+), or just "{METHOD}" otherwise. Wrap routes
// with Route to set the pattern name and the http.route attribute on any router.
mux.Handle("GET /api/hello", telemetry.Route("GET /api/hello", helloHandler))

// Client — wraps transport with automatic span + context propagation
client := telemetry.HTTPClient()
resp, err := client.Get("https://api.example.com/data")
```

#### Manual Tracing

```go
// Start a span
ctx, end := telemetry.StartSpan(ctx, "operation-name")
defer end()

// Record errors
telemetry.RecordError(ctx, err)

// Add attributes
telemetry.SetAttributes(ctx, attribute.String("order.id", orderID))

// Get trace ID for log correlation
traceID := telemetry.TraceIDFromContext(ctx)
```

#### Custom Metrics

`Init` installs the global `MeterProvider`; `Meter()` is the single access point services
should use to build their own instruments, so provider-level changes (views, cardinality
limits, common attributes) reach every service without code changes. Instrument types,
options and attributes come from the OpenTelemetry API packages (`go.opentelemetry.io/otel/metric`
and `go.opentelemetry.io/otel/attribute`), just like `attribute` is used with spans.

```go
// Name the scope after the importing module or package
meter := telemetry.Meter("github.com/delivery-much/my-service")

requests, err := meter.Int64Counter("my_service.orders.requests",
    metric.WithDescription("Total number of order requests"),
    metric.WithUnit("{request}"),
)

// Durations in seconds, with the same buckets as http.server.request.duration
duration, err := meter.Float64Histogram("my_service.orders.process.duration",
    metric.WithDescription("Time spent processing an order"),
    metric.WithUnit("s"),
    telemetry.WithHTTPDurationBuckets(),
)

requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "success")))
duration.Record(ctx, elapsed.Seconds(), metric.WithAttributes(attribute.String("result", "success")))
```

Before `Init` runs, or when telemetry is disabled, the returned meter is a no-op.
Keep attribute cardinality low: avoid user-controlled values (IDs, free-form headers)
as attributes — see `MetricCardinalityLimit` in Configuration.

### RabbitMQ

Package to connect, publish and consume queues in RabbitMQ, built on top of [rabbitmq/amqp091-go](https://github.com/rabbitmq/amqp091-go) (the maintained fork of the archived `streadway/amqp`).

Consume example:

```go
client, err := rabbitmq.New("amqp://guest:guest@localhost:5672/", "my-service")
if err != nil {
    panic(err)
}
defer client.Close()

err = client.Subscribe(rabbitmq.ConsumerConfig{
    ExchangeName: "orders",
    ExchangeType: "topic",
    QueueName:    "orders.created",
    BindingKey:   "orders.created",
    ConsumerName: "my-service",
}, func(ctx context.Context, msg *rabbitmq.Message) error {
    // business logic here
    return nil
})
```

Publish example:

```go
err = client.Publish(ctx, "orders", "orders.created", rabbitmq.Publishing{
    ContentType: "application/json",
    Body:        payload,
})
```

`Publish` sends to an exchange that must already exist (declared by a `Subscribe` on any service). All publishes share a single channel that is opened lazily and reopened when needed. `rabbitmq.Publishing` is an alias of `amqp091.Publishing`, so headers, delivery mode, message id and so on are available without importing the driver.

**Manual ack:** by default messages are auto-acked. Set `Consume.ManualAck` to decide per message from the handler result: `nil` acks, an error wrapped with `rabbitmq.Requeue` nacks with requeue (the broker redelivers it, flagged `Redelivered`), and any other error nacks without requeue (dead-lettered when the queue has a DLX, dropped otherwise). Use `Consume.PrefetchCount` to bound in-flight messages.

```go
err = client.Subscribe(rabbitmq.ConsumerConfig{
    ExchangeName: "orders",
    ExchangeType: "topic",
    QueueName:    "orders.created",
    BindingKey:   "orders.created",
    ConsumerName: "my-service",
    Consume:      rabbitmq.ConsumeOptions{ManualAck: true, PrefetchCount: 10},
    Queue:        rabbitmq.QueueOptions{Args: rabbitmq.Table{"x-dead-letter-exchange": "orders.dlx"}},
    HandlerTimeout: 30 * time.Second, // zero: 10s, negative: no timeout
}, func(ctx context.Context, msg *rabbitmq.Message) error {
    if err := process(ctx, msg.Body); errors.Is(err, ErrTemporary) {
        return rabbitmq.Requeue(err) // try again later
    } else if err != nil {
        return err // reject
    }
    return nil // ack
})
```

`ExchangeOptions`, `QueueOptions` and `ConsumeOptions` expose the remaining AMQP flags (auto-delete, exclusive, internal, no-wait, no-local and arguments). Exchanges and queues are durable by default; set `Durable: rabbitmq.NotDurable` on either option to declare them transient.

**Middlewares:** a `rabbitmq.Middleware` wraps a `SubscribeHandler`, so it can replace the context (tracing spans, log fields), inspect the delivery and observe the handler result (metrics). Client-level middlewares from `WithMiddlewares` run first, then the consumer's `Middlewares`, in the order given.

```go
tracing := func(next rabbitmq.SubscribeHandler) rabbitmq.SubscribeHandler {
    return func(ctx context.Context, msg *rabbitmq.Message) error {
        ctx, span := telemetry.StartSpan(ctx, "consume "+msg.Delivery.RoutingKey)
        defer span.End()
        return next(ctx, msg)
    }
}
client, err := rabbitmq.New(uri, "my-service", rabbitmq.WithMiddlewares(tracing))
```

**Reconnection:** automatic reconnection is enabled by default. When the broker connection drops, the client retries `DefaultReconnectMaxRetries` times with `DefaultReconnectInterval` between attempts, and re-declares exchanges, queues, bindings and consumers registered through `Subscribe` once the connection is back. Subscribe handlers keep running on the same goroutine, and `Publish` returns `amqp091.ErrClosed` while the reconnection is in progress (callers should retry). Lifecycle transitions (lost, recovered, closed) are logged. When every retry fails, the client stays closed: `Ping` returns `amqp091.ErrClosed` and consumers stop.

```go
// retry 20 times, 3s apart
client, err := rabbitmq.New(uri, "my-service", rabbitmq.WithReconnection(20, 3*time.Second))

// never give up: keep retrying every 5s until the broker is back
client, err := rabbitmq.New(uri, "my-service", rabbitmq.WithReconnection(rabbitmq.ReconnectForever, 5*time.Second))

// legacy behaviour: no reconnection
client, err := rabbitmq.New(uri, "my-service", rabbitmq.WithoutReconnection())
```

**Integration tests:** the `rabbitmq` package has integration tests that run only when a broker is available. `docker compose up -d rabbitmq` starts one with the management plugin, then:

```bash
RABBITMQ_URI=amqp://guest:guest@localhost:5672/ \
RABBITMQ_MANAGEMENT_URL=http://guest:guest@localhost:15672 \
go test ./rabbitmq/ -run Integration -v
```

`RABBITMQ_MANAGEMENT_URL` is only needed by the reconnection test, which drops the broker connection through the management API.

**Migration note (streadway/amqp -> amqp091-go):** `Message.Delivery` is now an `amqp091.Delivery`. The API is identical, so services only need to replace the import `github.com/streadway/amqp` by `github.com/rabbitmq/amqp091-go` wherever they reference that type directly.
