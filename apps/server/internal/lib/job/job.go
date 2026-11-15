package job

import (
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/lib/email"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/lib/redisconfig"
)

type JobService struct {
	client      *asynq.Client
	server      *asynq.Server
	logger      *slog.Logger
	emailClient *email.Client
}

func NewJobService(logger *slog.Logger, cfg *config.Config, redisOptions redisconfig.Options) *JobService {
	client := asynq.NewClient(redisOptions.Asynq())

	server := asynq.NewServer(
		redisOptions.Asynq(),
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6, // Higher priority queue for important emails
				"default":  3, // Default priority for most emails
				"low":      1, // Lower priority for non-urgent emails
			},
		},
	)

	return &JobService{
		client:      client,
		server:      server,
		logger:      logger,
		emailClient: email.NewClient(cfg, logger),
	}
}

func (j *JobService) Start() error {
	// Register task handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskWelcome, j.handleWelcomeEmailTask)

	j.logger.Info("Starting background job server")
	if err := j.server.Start(mux); err != nil {
		return err
	}

	return nil
}

func (j *JobService) Stop() {
	j.logger.Info("Stopping background job server")
	j.server.Shutdown()
	if err := j.client.Close(); err != nil {
		j.logger.Error("close asynq client", slog.Any("error", err))
	}
}
