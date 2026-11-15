package middleware

import (
	"context"
	"testing"
)

func TestTracingAccessorsUseRequestAndAuthContextValues(t *testing.T) {
	ctx := WithRequestID(context.Background(), "request-123")
	ctx = WithUserID(ctx, "user-123")

	if got := RequestID(ctx); got != "request-123" {
		t.Fatalf("RequestID() = %q, want %q", got, "request-123")
	}
	if got := GetRequestID(ctx); got != "request-123" {
		t.Fatalf("GetRequestID() = %q, want %q", got, "request-123")
	}
	if got := TracingUserID(ctx); got != "user-123" {
		t.Fatalf("TracingUserID() = %q, want %q", got, "user-123")
	}
	if got, ok := UserID(ctx); !ok || got != "user-123" {
		t.Fatalf("UserID() = (%q, %t), want (%q, true)", got, ok, "user-123")
	}
}
