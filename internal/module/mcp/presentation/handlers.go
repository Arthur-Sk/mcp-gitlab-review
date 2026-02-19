package presentation

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	mcpService "mcp-gitlab-review/internal/module/mcp/service"
	"mcp-gitlab-review/internal/platform/config"
)

type Handler struct {
	service *mcpService.Service
	config  *config.Config
	logger  *zap.Logger
}

func NewHandler(service *mcpService.Service, cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		config:  cfg,
		logger:  logger,
	}
}

func (h *Handler) NewMCPServer() *server.MCPServer {
	srv := server.NewMCPServer(
		h.config.MCP.Name,
		h.config.MCP.Version,
	)

	srv.AddTool(getMRSummaryTool(), h.handleGetMRSummary)
	srv.AddTool(getChangedFilesTool(), h.handleGetChangedFiles)
	srv.AddTool(getFileContextTool(), h.handleGetFileContext)
	srv.AddTool(getArchitectureViewTool(), h.handleGetArchitectureView)
	srv.AddTool(postReviewCommentTool(), h.handlePostReviewComment)

	return srv
}

func (h *Handler) handleGetMRSummary(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.logger.Info("get_mr_summary called", zap.String("mr_url", mrURL))

	result, err := h.service.GetMRSummary(ctx, mrURL)
	if err != nil {
		h.logger.Error("get_mr_summary failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *Handler) handleGetChangedFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.logger.Info("get_changed_files called", zap.String("mr_url", mrURL))

	result, err := h.service.GetChangedFiles(ctx, mrURL)
	if err != nil {
		h.logger.Error("get_changed_files failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *Handler) handleGetFileContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.logger.Info("get_file_context called",
		zap.String("mr_url", mrURL),
		zap.String("file_path", filePath),
	)

	result, err := h.service.GetFileContext(ctx, mrURL, filePath)
	if err != nil {
		h.logger.Error("get_file_context failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *Handler) handleGetArchitectureView(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	path := request.GetString("path", "")

	h.logger.Info("get_architecture_view called",
		zap.String("mr_url", mrURL),
		zap.String("path", path),
	)

	result, err := h.service.GetArchitectureView(ctx, mrURL, path)
	if err != nil {
		h.logger.Error("get_architecture_view failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *Handler) handlePostReviewComment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body, err := request.RequireString("body")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	filePath := request.GetString("file_path", "")
	newLine := intPtrFromFloat(request.GetFloat("new_line", 0))
	oldLine := intPtrFromFloat(request.GetFloat("old_line", 0))

	h.logger.Info("post_review_comment called",
		zap.String("mr_url", mrURL),
		zap.String("file_path", filePath),
	)

	commentReq := mcpService.PostCommentRequest{
		FilePath: filePath,
		NewLine:  newLine,
		OldLine:  oldLine,
		Body:     body,
	}

	if err := h.service.PostReviewComment(ctx, mrURL, commentReq); err != nil {
		h.logger.Error("post_review_comment failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(`{"status": "comment posted successfully"}`), nil
}

func intPtrFromFloat(val float64) *int {
	if val == 0 {
		return nil
	}

	intVal := int(val)

	return &intVal
}
