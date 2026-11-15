package job

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/lib/email"
)

func (j *JobService) InitHandlers(config *config.Config, logger *slog.Logger) {
	j.emailClient = email.NewClient(config, logger)
}

func (j *JobService) handleWelcomeEmailTask(ctx context.Context, t *asynq.Task) error {
	var p WelcomeEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal welcome email payload: %w", err)
	}

	j.logger.InfoContext(ctx, "processing welcome email task", slog.String("task", TaskWelcome))

	if err := j.emailClient.SendWelcomeEmail(p.To, p.FirstName); err != nil {
		return fmt.Errorf("send welcome email: %w", err)
	}

	j.logger.InfoContext(ctx, "Successfully sent welcome email", slog.String("task", TaskWelcome))
	return nil
}
