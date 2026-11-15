package middleware

import (
	"context"
	"net/http"
	"uuid"
)

const (
	RequestIdHeader = "X-Request-ID"
)

type RequestContextKey struct{}

func RequestId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIdHeader)
		if !validRequestId(requestID) {
			requestID = uuid.New().String()
		}

		w.Header().Set(RequestIdHeader, requestID)

		ctx := context.WithValue(r.Context(), RequestContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	if requestId, ok := ctx.Value(RequestContextKey{}).(string); ok {
		return requestId
	}
	return ""
}

func validRequestId(str string) bool {
	if len(str) == 0 || len(str) > 64 {
		return false
	}
	for _, r := range str {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
