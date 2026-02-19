package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/diff"
	"mcp-gitlab-review/internal/module/gitlab"
	mcpModule "mcp-gitlab-review/internal/module/mcp"
	"mcp-gitlab-review/internal/module/mcp/presentation"
	"mcp-gitlab-review/internal/platform/cache"
	"mcp-gitlab-review/internal/platform/config"
	"mcp-gitlab-review/internal/platform/logger"
)

var Module = fx.Options(
	fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		return &fxevent.ZapLogger{Logger: log}
	}),
	config.Module,
	logger.Module,
	cache.Module,
	gitlab.Module,
	diff.Module,
	mcpModule.Module,
	fx.Invoke(run),
)

func run(lc fx.Lifecycle, handler *presentation.Handler, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			mcpServer := handler.NewMCPServer()
			stdioServer := server.NewStdioServer(mcpServer)

			log.Info("starting MCP server via stdio")

			go func() {
				sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				defer stop()

				if err := stdioServer.Listen(sigCtx, os.Stdin, os.Stdout); err != nil {
					log.Error("MCP server error", zap.Error(err))
				}
			}()

			return nil
		},
	})
}
