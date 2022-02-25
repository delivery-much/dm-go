package middleware

import (
	"context"
	"net/http"

	"github.com/delivery-much/dm-go/logger"
	"github.com/google/uuid"
)

// Key to use when setting the request ID.
type ctxKeyRequestID int

// RequestIDKey is the key that holds the unique request ID in a request context.
const RequestIDKey ctxKeyRequestID = 0

// RequestID is a middleware that injects a request ID into the context and logger of each
// request. A request ID is an UUID, example: 9e21998d-d36f-48ef-831b-30e643536c88.
func RequestID(headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := r.Header.Get(getHeaderName(headerName))
			if requestID == "" {
				requestID = uuid.New().String()
			}

			ctx = context.WithValue(ctx, RequestIDKey, requestID)

			if logger.Instantiated() {
				logger.AddRequestID(requestID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// getHeaderName returns the key of header. Default: X-Request-Id.
func getHeaderName(headerName string) string {
	if headerName != "" {
		return headerName
	}
	return "X-Request-Id"
}

// GetReqID returns a request ID from the given context if one is present.
// Returns the empty string if a request ID cannot be found.
func GetReqID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}
