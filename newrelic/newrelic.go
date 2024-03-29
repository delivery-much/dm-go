package newrelic

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	newRelicApp *newrelic.Application
)

// InitNewRelic init the client application of New Relic, for metrics and monitoring
func InitNewRelic(appName string, key string) {
	var err error
	newRelicApp, err = newrelic.NewApplication(
		newrelic.ConfigAppName(strings.ToLower(appName)),
		newrelic.ConfigLicense(key),
	)

	if err != nil {
		log.Printf("Cant init New Relic agent: %v", err)
	}
}

// WrapHandleFunc instruments handler functions using transactions. To record route/handler transactions in New Relic
func WrapHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	return newrelic.WrapHandleFunc(newRelicApp, pattern, handler)
}

func NewRelicGinMiddleware() gin.HandlerFunc {
	return nrgin.Middleware(newRelicApp)
}

// StartNewRelicCustomSegment creates a custom segment for metrics/monitoring, by passing the transaction context and a custom name
func StartNewRelicCustomSegment(ctx context.Context, name string) context.CancelFunc {
	txn := newrelic.FromContext(ctx)
	s := txn.StartSegment(name)
	return func() { s.End() }
}

// StartNewRelicDBSegment creates a SQL database segment for metrics/monitoring, by passing the transaction context and operation (select, insert)
func StartNewRelicDBSegment(ctx context.Context, operation string, collection string) context.CancelFunc {
	txn := newrelic.FromContext(ctx)
	s := newrelic.DatastoreSegment{
		StartTime:  txn.StartSegmentNow(),
		Product:    newrelic.DatastoreMySQL,
		Collection: collection,
		Operation:  operation,
	}
	return func() { s.End() }
}
