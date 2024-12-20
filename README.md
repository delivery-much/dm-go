<p align="center"><img src="assets/gopher.png" width="350"></p>

<h1 align="center">
  dm-go
</h1>

* [Overview](#overview)
* [Packages](#packages)
	* [Logger](#logger)
	* [Middleware](#middleware)
	* [New Relic](#new-relic)
	* [Render](#render)
	* [String Utils](#string-utils)
	* [Open Telemetry](#open-telemetry)
	* [Request](#request)

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

 - SetMonitor(\*options.ClientOptions): creates a new \*event.CommandMonitor and sets as a monitor to the clientOptions, only *if* there is a openTelemetry trace exporter created

Example:
```golang
c := &Client{}
clientOptions := options.Client().
 ApplyURI(uri)
optel.SetMonitor(clientOptions)
c.conn, err = mongo.Connect(ctx, clientOptions)
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
