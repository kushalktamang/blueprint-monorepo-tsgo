package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/errs"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
)

type AuthContextKey string

const (
	userIDKey      AuthContextKey = "user_id"
	userRoleKey    AuthContextKey = "user_role"
	permissionsKey AuthContextKey = "permissions"
)

func UserID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(userIDKey).(string)
	return value, ok
}

func UserRole(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(userRoleKey).(string)
	return value, ok
}

func Permissions(ctx context.Context) ([]string, bool) {
	value, ok := ctx.Value(permissionsKey).([]string)
	return value, ok
}

type AuthMiddleware struct {
	server *server.Server
}

func NewAuthMiddleware(server *server.Server) *AuthMiddleware {
	return &AuthMiddleware{
		server: server,
	}
}

func (auth *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	authenticated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		claims, ok := clerk.SessionClaimsFromContext(ctx)
		if !ok {
			auth.server.Logger.ErrorContext(ctx, "could not get session claims from context",
				slog.String("function", "RequireAuth"),
				slog.String("request_id", GetRequestID(ctx)),
				slog.Duration("duration", time.Since(start)))
			auth.writeUnauthorized(w, r)
			return
		}

		ctx = context.WithValue(ctx, userIDKey, claims.Subject)
		ctx = context.WithValue(ctx, userRoleKey, claims.ActiveOrganizationRole)
		ctx = context.WithValue(ctx, permissionsKey, claims.Claims.ActiveOrganizationPermissions)

		auth.server.Logger.InfoContext(ctx, "user authenticated successfully",
			slog.String("function", "RequireAuth"),
			slog.String("user_id", claims.Subject),
			slog.String("request_id", GetRequestID(ctx)),
			slog.Duration("duration", time.Since(start)))

		next.ServeHTTP(w, r.WithContext(ctx))
	})

	return clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(auth.writeUnauthorized)),
	)(authenticated)
}

func (auth *AuthMiddleware) writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	err := json.NewEncoder(w).Encode(errs.HTTPError{
		Code:     "UNAUTHORIZED",
		Message:  http.StatusText(http.StatusUnauthorized),
		Override: false,
		Status:   http.StatusUnauthorized,
	})
	if err != nil {
		auth.server.Logger.ErrorContext(r.Context(), "failed to write JSON response",
			slog.String("function", "RequireAuth"), slog.Any("error", err))
	}
}
