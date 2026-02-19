package diff

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/diff/service"
)

var Module = fx.Module("diff",
	fx.Decorate(func(log *zap.Logger) *zap.Logger { return log.Named("diff") }),
	fx.Provide(service.NewParser),
)
