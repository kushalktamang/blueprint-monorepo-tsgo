package router

import (
	"net/http"
	"slices"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/handler"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/middleware"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/service"
)

type Middleware = func(http.Handler) http.Handler

// chain applies mws so that the first one is the outermost.
func chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := range slices.Backward(mws) {
		h = mws[i](h)
	}
	return h
}

func NewRouter(s *server.Server, h *handler.Handlers, services *service.Services) http.Handler {
	mw := middleware.NewMiddlewares(s)

	mux := http.NewServeMux()
	registerSystemRoutes(mux, h)

	v1 := http.NewServeMux()
	// Register application routes on v1 as they are added. The fallback keeps
	// the empty API surface explicit instead of returning an ambiguous response.
	v1.Handle("/", http.NotFoundHandler())
	protectedAPI := mw.Auth.RequireAuth(v1)
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", protectedAPI))

	limiter := middleware.NewRateLimitMiddleware(s, 20, 20)
	s.RegisterCleanup(limiter.Close)

	return chain(mux,
		middleware.RequestId, // first, so everything below can log it
		mw.Global.Recover,    // outside the logger so panics in later middleware are caught
		limiter.Handler,
		mw.Global.CORS,
		mw.Global.Secure,
		mw.Tracing.NewRelic,               // creates the txn
		mw.Tracing.EnhanceTracing,         // needs request ID + txn
		mw.ContextEnhancer.EnhanceContext, // needs txn for trace context
		mw.Global.RequestLogger,
	)
}
