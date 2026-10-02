package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// HTTPDurationBucketBoundaries are the explicit histogram bucket boundaries, in seconds,
// recommended by the OpenTelemetry semantic conventions for HTTP request durations.
// They are the same boundaries used by the Middleware on http.server.request.duration,
// so custom duration histograms built with them stay comparable to the standard one.
var HTTPDurationBucketBoundaries = []float64{
	0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
}

// Meter returns a named meter from the global MeterProvider.
//
// Use the importing package (or module) path as the name, so the instruments are grouped
// under a meaningful instrumentation scope. Before Init installs a provider, or when
// telemetry is disabled, the returned meter is a no-op.
//
// Example:
//
//	meter := telemetry.Meter("github.com/delivery-much/my-service")
//	requests, err := meter.Int64Counter("my_service.requests",
//	    metric.WithDescription("Total number of requests"),
//	    metric.WithUnit("{request}"),
//	)
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

// WithHTTPDurationBuckets configures a histogram with HTTPDurationBucketBoundaries.
// Record values in seconds when using it.
//
// Example:
//
//	duration, err := meter.Float64Histogram("my_service.process.duration",
//	    metric.WithUnit("s"),
//	    telemetry.WithHTTPDurationBuckets(),
//	)
func WithHTTPDurationBuckets() metric.HistogramOption {
	return metric.WithExplicitBucketBoundaries(HTTPDurationBucketBoundaries...)
}
