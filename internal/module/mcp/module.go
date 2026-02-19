package mcp

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/mcp/presentation"
	"mcp-gitlab-review/internal/module/mcp/service"
)

var Module = fx.Module("mcp",
	fx.Decorate(func(log *zap.Logger) *zap.Logger { return log.Named("mcp") }),
	fx.Provide(
		service.NewService,
		presentation.NewHandler,
	),
)
