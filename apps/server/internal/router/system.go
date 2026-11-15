package router

import (
	"io/fs"
	"net/http"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/assets"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/handler"
)

func registerSystemRoutes(mux *http.ServeMux, h *handler.Handlers) {
	mux.HandleFunc("GET /status", h.Health.CheckHealth)
	mux.HandleFunc("GET /docs", h.OpenAPI.ServerOpenApiUi)

	staticFS, err := fs.Sub(assets.FS, "static")
	if err != nil {
		panic("embedded static assets are unavailable: " + err.Error())
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
}
