package repository

import "github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"

type Repositories struct{}

func NewRepositories(s *server.Server) *Repositories {
	return &Repositories{}
}
