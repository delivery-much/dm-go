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

### Open Telemetry

Package with functions used for OpenTelemetry monitoring, from start a new OpenTelemetry connection to setup requests and database tracing.

#### Setting up the connection 
 - StartOptelConnection(ctx context.Context, c OptelConfiguration) (err error): starts a connection to a OpenTelemetry collector, and set up all the essential resources to monitor and export tracing.
 - ShutdownOptelConnection() (err error): calls the cleanup process that shutsdown the OpenTelemetry components. It should be called before finishing the application (using defer is recommended) 
 
Example:
```golang
err := StartOptelConnection(ctx, myConfig)
    if err != nil {
      handleErrorFunction(err)
    }
	defer ShutdownOptelConnection()
```

#### Implementing traces
 - TraceMiddlewares(appName string, r chi.Routes) (middlewares []func(next http.Handler) http.Handler): Returns a slice of chi.middlewares capable of tracing general http information, as well as the dm-go/middleware/request_id. All of the information is acquired through the context.
```golang
r := chi.NewRouter()
r.Use(optel.TraceMiddlewares("rochelle-coupon", r)...)
```

 - StartTrack(ctx context.Context, n string) func(): Starts a new span (a Tracing 'checkpoint') with the name 'n', it is essential to call the End function returned by the StartTrack function, preferably using `defer`

Example:
```golang
defer StartTrack(ctx, "myEventName")()
```

Verbose example:
```golang
endFunction := StartTrack(ctx, "myEventName")
defer endFunction()
```


#### MongoDB Traces
 - NewMongoMonitor() \*event.CommandMonitor: returns a new \*event.CommandMonitor for the mongodb client. The returned event monitor is meant to be set in the mongoDB clientOptions through the mongo.NewMonitor function

Example:
```golang
c := &Client{}
clientOptions := options.Client().
 ApplyURI(uri)
clientOptions.SetMonitor(optel.NewMongoMonitor())
c.conn, err = mongo.Connect(ctx, clientOptions)
```

