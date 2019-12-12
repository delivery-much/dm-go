package newrelic

import (
	"context"
	"log"
	"net/http"
	"strings"

	newrelic "github.com/newrelic/go-agent"
)

var (
	newRelicApp newrelic.Application
)

// InitNewRelic init the client application of New Relic, for metrics and monitoring
func InitNewRelic(appName string, key string) {
	var err error
	config := newrelic.NewConfig(strings.ToLower(appName), key)
	newRelicApp, err = newrelic.NewApplication(config)

	if err != nil {
		log.Printf("Cant init New Relic agent: %v", err)
	}
}

// WrapHandleFunc instruments handler functions using transactions. To record route/handler transactions in New Relic
func WrapHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	return newrelic.WrapHandleFunc(newRelicApp, pattern, handler)
}

// StartNewRelicCustomSegment creates a custom segment for metrics/monitoring, by passing the transaction context and a custom name
func StartNewRelicCustomSegment(ctx context.Context, name string) context.CancelFunc {
	txn := newrelic.FromContext(ctx)
	s := newrelic.StartSegment(txn, name)
	return func() { s.End() }
}

// StartNewRelicDBSegment creates a SQL database segment for metrics/monitoring, by passing the transaction context and operation (select, insert)
func StartNewRelicDBSegment(ctx context.Context, operation string, collection string) context.CancelFunc {
	txn := newrelic.FromContext(ctx)
	s := newrelic.DatastoreSegment{
		StartTime:  newrelic.StartSegmentNow(txn),
		Product:    newrelic.DatastoreMySQL,
		Collection: collection,
		Operation:  operation,
	}
	return func() { s.End() }
}
