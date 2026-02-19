package cache

import (
	"go.uber.org/fx"

	"mcp-gitlab-review/internal/platform/config"
)

var Module = fx.Module("cache",
	fx.Provide(func(cfg *config.Config) *Cache {
		return New(cfg.Cache.TTL, cfg.Cache.MaxSize)
	}),
)
