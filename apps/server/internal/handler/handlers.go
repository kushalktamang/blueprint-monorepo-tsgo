package handler

import (
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/service"
)

type Handlers struct {
	Health  *HealthHandler
	OpenAPI *OpenApiHandler
}

func NewHandlers(s *server.Server, services *service.Services) *Handlers {
	return &Handlers{
		Health:  NewHealthHandler(s),
		OpenAPI: NewOpenApiHandler(s),
	}
}
