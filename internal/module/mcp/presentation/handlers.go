package presentation

import (
	"mcp-gitlab-review/internal/module/mcp/service/tools"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/platform/config"
)

type Handler struct {
	tools  []tools.ToolHandler
	config *config.Config
	logger *zap.Logger
}

type HandlerParams struct {
	fx.In

	Tools  []tools.ToolHandler `group:"mcp_tools"`
	Config *config.Config
	Logger *zap.Logger
}

func NewHandler(params HandlerParams) *Handler {
	return &Handler{
		tools:  params.Tools,
		config: params.Config,
		logger: params.Logger,
	}
}

func (h *Handler) NewMCPServer() *server.MCPServer {
	srv := server.NewMCPServer(
		h.config.MCP.Name,
		h.config.MCP.Version,
	)

	for _, tool := range h.tools {
		srv.AddTool(tool.Tool(), tool.Handle)
	}

	return srv
}
