package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID(t *testing.T) {
	// create a handler to use as "next" which will verify the request
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := GetReqID(r.Context())
		if reqID == "" {
			t.Fatalf("Request ID must be not nil, but got nil")
		}
	})

	next := RequestID("")
	handlerToTest := next(nextHandler)

	// create a mock request to use
	req := httptest.NewRequest(http.MethodPost, "http://localhost", nil)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
}

func TestRequestIDInHeader(t *testing.T) {
	headerName := "Request-Id"
	uuid := "82e800e3-1cab-4b28-ac01-11989db21b55"

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := GetReqID(r.Context())
		if reqID != uuid {
			t.Fatalf("Request ID must be equal to %s but got %s", uuid, reqID)
		}
	})

	next := RequestID(headerName)
	handlerToTest := next(nextHandler)

	// create a mock request to use
	req := httptest.NewRequest(http.MethodPost, "http://localhost", nil)
	req.Header.Set(headerName, uuid)

	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
}
