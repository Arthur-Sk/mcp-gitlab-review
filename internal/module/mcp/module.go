package mcp

import (
	"mcp-gitlab-review/internal/module/mcp/service"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/mcp/presentation"
)

var Module = fx.Module("mcp",
	fx.Decorate(func(log *zap.Logger) *zap.Logger { return log.Named("mcp") }),
	service.Module,
	fx.Provide(
		presentation.NewHandler,
	),
)
