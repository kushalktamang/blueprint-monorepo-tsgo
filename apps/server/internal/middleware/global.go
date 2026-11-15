package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/errs"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
	"github.com/rs/cors"
)

type GlobalMiddlewares struct {
	server *server.Server
}

func NewGlobalMiddlewares(s *server.Server) *GlobalMiddlewares {
	return &GlobalMiddlewares{
		server: s,
	}
}

func (global *GlobalMiddlewares) CORS(next http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: global.server.Config.Server.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPut,
			http.MethodPatch, http.MethodPost, http.MethodDelete,
		},
		AllowedHeaders: []string{"Authorization", "Content-Type", RequestIdHeader},
	}).Handler(next)
}

// statusRecorder captures the status code so it can be logged after the handler runs.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach Flusher, Hijacker, etc.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
func (global *GlobalMiddlewares) RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		level := slog.LevelInfo
		switch {
		case rec.status >= 500:
			level = slog.LevelError
		case rec.status >= 400:
			level = slog.LevelWarn
		}

		ctx := r.Context()
		global.server.Logger.Log(ctx, level, "API",
			slog.String("request_id", GetRequestID(ctx)),
			slog.Duration("latency", time.Since(start)),
			slog.Int("status", rec.status),
			slog.String("method", r.Method),
			slog.String("uri", r.URL.RequestURI()),
			slog.String("host", r.Host),
			slog.String("ip", clientIp(r)),
			slog.String("user_agent", r.UserAgent()),
		)
	})
}

func (global *GlobalMiddlewares) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec) // intentional abort, let net/http handle it
			}

			global.server.Logger.ErrorContext(r.Context(), "panic recovered",
				slog.String("request_id", GetRequestID(r.Context())),
				slog.Any("panic", rec),
				slog.String("stack", string(debug.Stack())))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(errs.HTTPError{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: http.StatusText(http.StatusInternalServerError),
				Status:  http.StatusInternalServerError,
			})
		}()
		next.ServeHTTP(w, r)
	})
}

func (global *GlobalMiddlewares) Secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}
