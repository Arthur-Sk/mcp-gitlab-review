package gitlab

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/gitlab/repository"
	"mcp-gitlab-review/internal/module/gitlab/service"
)

var Module = fx.Module("gitlab",
	fx.Decorate(func(log *zap.Logger) *zap.Logger { return log.Named("gitlab") }),
	fx.Provide(
		repository.NewClient,
		service.NewService,
	),
)
