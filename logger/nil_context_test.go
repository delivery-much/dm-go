package logger

import (
	"context"
	"testing"
)

// TestPackageFunctionsAcceptNilContext guards against a nil pointer dereference
// in addCTXFields when callers pass a nil context to the package-level log functions.
func TestPackageFunctionsAcceptNilContext(t *testing.T) {
	disableOTelLogsByEnv(t)
	t.Cleanup(func() { log = nil })

	if err := NewLogger(Configuration{IsJSON: true, Level: DEBUG}); err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	var ctx context.Context // intentionally nil

	Debug(ctx, "debug")
	Debugw(ctx, "debugw", "k", "v")
	Debugf(ctx, "debugf %d", 1)
	Info(ctx, "info")
	Infow(ctx, "infow", "k", "v")
	Infof(ctx, "infof %d", 1)
	Warn(ctx, "warn")
	Warnw(ctx, "warnw", "k", "v")
	Warnf(ctx, "warnf %d", 1)
	Error(ctx, "error")
	Errorw(ctx, "errorw", "k", "v")
	Errorf(ctx, "errorf %d", 1)
}
