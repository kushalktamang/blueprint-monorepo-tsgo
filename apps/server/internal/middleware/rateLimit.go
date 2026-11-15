package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimitMiddleware struct {
	server   *server.Server
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
	done     chan struct{}
	once     sync.Once
}

func NewRateLimitMiddleware(s *server.Server, rps float64, burst int) *RateLimitMiddleware {
	rl := &RateLimitMiddleware{
		server:   s,
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
		done:     make(chan struct{}),
	}
	go rl.cleanup(time.Minute)
	return rl
}

// Close stops the cleanup goroutine. Safe to call more than once.
func (rl *RateLimitMiddleware) Close() {
	rl.once.Do(func() { close(rl.done) })
}

func (rl *RateLimitMiddleware) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

// cleanup evicts idle visitors so the map doesn't grow forever.
func (rl *RateLimitMiddleware) cleanup(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-rl.done:
			return
		case <-t.C:
			rl.mu.Lock()
			for k, v := range rl.visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, k)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIp(r)
		if rl.allow(ip) {
			next.ServeHTTP(w, r)
			return
		}

		rl.RecordRateLimitHit(r.URL.Path)
		rl.server.Logger.WarnContext(r.Context(), "rate limit exceeded",
			slog.String("request_id", RequestID(r.Context())),
			slog.String("remote_addr", ip),
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method),
		)

		w.Header().Set("Retry-After", strconv.Itoa(1))
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
	})
}

func (rl *RateLimitMiddleware) RecordRateLimitHit(endpoint string) {
	ls := rl.server.LoggerService
	if ls == nil || ls.GetApplication() == nil {
		return
	}
	ls.GetApplication().RecordCustomEvent("RateLimitHit", map[string]any{
		"endpoint": endpoint,
	})
}
