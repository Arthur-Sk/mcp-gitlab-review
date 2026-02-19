package service

import (
	"mcp-gitlab-review/internal/module/mcp/service/tools"

	"go.uber.org/fx"
)

var Module = fx.Module("mcp.tools",
	fx.Provide(
		asToolHandler(tools.NewGetMRSummaryHandler),
		asToolHandler(tools.NewGetChangedFilesHandler),
		asToolHandler(tools.NewGetFileContextHandler),
		asToolHandler(tools.NewGetArchitectureViewHandler),
		asToolHandler(tools.NewPostReviewCommentHandler),
	),
)

func asToolHandler(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(tools.ToolHandler)),
		fx.ResultTags(`group:"mcp_tools"`),
	)
}
