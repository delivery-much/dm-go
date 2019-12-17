# dm-go

Reusable packages and frameworks for Go services

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

logger.Infow("failed to fetch URL",
    // Key and value after the message
    "url", "www.google.com",
    "attempt", 3,
    "backoff", time.Second,
)
```

### New Relic

Package with some helpers/middleware for send events to New Relic.

Example:

```go
InitNewRelic("app-teste", "1234")

// Func for database monitoring
newRelicSegment := StartNewRelicDBSegment(ctx.Background(), "select", "user")
defer newRelicSegment()

// Middleware for instruments handler functions using transactions and monitoring the route 
router.Post(newrelic.WrapHandleFunc("/test", h.handleTest()))
```

### Render

Package with some helpers for render responses. To respond in JSON format.