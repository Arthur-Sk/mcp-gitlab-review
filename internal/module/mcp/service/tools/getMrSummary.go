package tools

import (
	"context"
	"fmt"
	"mcp-gitlab-review/internal/module/mcp/service/helper"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
	gitlabService "mcp-gitlab-review/internal/module/gitlab/service"
)

type MRSummaryResponse struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Author         string   `json:"author"`
	SourceBranch   string   `json:"source_branch"`
	TargetBranch   string   `json:"target_branch"`
	State          string   `json:"state"`
	WebURL         string   `json:"web_url"`
	HasConflicts   bool     `json:"has_conflicts"`
	ChangesCount   string   `json:"changes_count"`
	CommitMessages []string `json:"commit_messages,omitempty"`
}

type GetMRSummaryHandler struct {
	gitlab *gitlabService.Service
	logger *zap.Logger
}

func NewGetMRSummaryHandler(gitlab *gitlabService.Service, logger *zap.Logger) *GetMRSummaryHandler {
	return &GetMRSummaryHandler{
		gitlab: gitlab,
		logger: logger,
	}
}

func (h *GetMRSummaryHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"get_mr_summary",
		mcp.WithDescription(
			"Get merge request metadata: title, description, author, branches, "+
				"commit messages, and status. Use this first to understand the MR intent "+
				"before diving into code changes.",
		),
		mcp.WithString(
			"mr_url",
			mcp.Required(),
			mcp.Description("Full GitLab merge request URL (e.g. https://gitlab.com/group/project/-/merge_requests/123)"),
		),
	)
}

func (h *GetMRSummaryHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.logger.Info("get_mr_summary called", zap.String("mr_url", mrURL))

	result, err := h.getMRSummary(ctx, mrURL)
	if err != nil {
		h.logger.Error("get_mr_summary failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *GetMRSummaryHandler) getMRSummary(ctx context.Context, mrURL string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := h.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR summary: %w", err)
	}

	response := MRSummaryResponse{
		Title:          mr.Title,
		Description:    mr.Description,
		Author:         fmt.Sprintf("%s (@%s)", mr.Author.Name, mr.Author.Username),
		SourceBranch:   mr.SourceBranch,
		TargetBranch:   mr.TargetBranch,
		State:          mr.State,
		WebURL:         mr.WebURL,
		HasConflicts:   mr.HasConflicts,
		ChangesCount:   mr.ChangesCount,
		CommitMessages: mr.CommitMessages,
	}

	return helper.ToJSON(response)
}
