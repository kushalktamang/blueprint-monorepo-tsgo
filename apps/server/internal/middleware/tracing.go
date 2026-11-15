package middleware

import (
	"context"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func TracingUserID(ctx context.Context) string {
	id, _ := UserID(ctx)
	return id
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestContextKey{}, id)
}

func RequestID(ctx context.Context) string {
	return GetRequestID(ctx)
}

type TracingMiddleware struct {
	nrApp *newrelic.Application
}

func NewTracingMiddleware(nrApp *newrelic.Application) *TracingMiddleware {
	return &TracingMiddleware{nrApp: nrApp}
}

// NewRelic starts a transaction per request and puts it in the context.
func (tm *TracingMiddleware) NewRelic(next http.Handler) http.Handler {
	if tm.nrApp == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn := tm.nrApp.StartTransaction(r.Method + " " + r.URL.Path)
		defer txn.End()

		txn.SetWebRequestHTTP(r)
		w = txn.SetWebResponse(w) // records the status code for you

		r = r.WithContext(newrelic.NewContext(r.Context(), txn))
		next.ServeHTTP(w, r)
	})
}

// EnhanceTracing adds custom attributes. Must run inside NewRelic.
func (tm *TracingMiddleware) EnhanceTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn := newrelic.FromContext(r.Context())
		if txn == nil {
			next.ServeHTTP(w, r)
			return
		}

		if id := RequestID(r.Context()); id != "" {
			txn.AddAttribute("request.id", id)
		}
		if id := TracingUserID(r.Context()); id != "" {
			txn.AddAttribute("user.id", id)
		}

		next.ServeHTTP(w, r)
	})
}
