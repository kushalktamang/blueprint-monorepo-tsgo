package handler

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/assets"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
)

type OpenApiHandler struct {
	Handler
}

func NewOpenApiHandler(s *server.Server) *OpenApiHandler {
	return &OpenApiHandler{
		Handler: NewHandler(s),
	}
}

func (h *OpenApiHandler) ServerOpenApiUi(w http.ResponseWriter, r *http.Request) {
	html, err := fs.ReadFile(assets.FS, "static/openapi.html")
	if err != nil {
		slog.Error("failed to read OpenApi Ui template", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(html); err != nil {
		slog.Error("failed to write OpenAPI UI response", "error", err)
	}
}
