package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
	gitlabService "mcp-gitlab-review/internal/module/gitlab/service"
)

type PostCommentRequest struct {
	FilePath string
	NewLine  *int
	OldLine  *int
	Body     string
}

type PostReviewCommentHandler struct {
	gitlab *gitlabService.Service
	logger *zap.Logger
}

func NewPostReviewCommentHandler(gitlab *gitlabService.Service, logger *zap.Logger) *PostReviewCommentHandler {
	return &PostReviewCommentHandler{
		gitlab: gitlab,
		logger: logger,
	}
}

func (h *PostReviewCommentHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"post_review_comment",
		mcp.WithDescription(
			"Post a review comment on the merge request. Can be a general comment "+
				"or a line-specific comment on a file. For line-specific comments, "+
				"provide file_path and either new_line (for additions) or old_line (for deletions).",
		),
		mcp.WithString(
			"mr_url",
			mcp.Required(),
			mcp.Description("Full GitLab merge request URL"),
		),
		mcp.WithString(
			"body",
			mcp.Required(),
			mcp.Description("The comment text (supports Markdown)"),
		),
		mcp.WithString(
			"file_path",
			mcp.Description("File path for line-specific comment (optional for general comments)"),
		),
		mcp.WithNumber(
			"new_line",
			mcp.Description("Line number in the new version of the file"),
		),
		mcp.WithNumber(
			"old_line",
			mcp.Description("Line number in the old version of the file"),
		),
	)
}

func (h *PostReviewCommentHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body, err := request.RequireString("body")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	decoratedBody := fmt.Sprintf("%s\n\n%s", "# AI Review", body)

	filePath := request.GetString("file_path", "")
	newLine := intPtrFromFloat(request.GetFloat("new_line", 0))
	oldLine := intPtrFromFloat(request.GetFloat("old_line", 0))

	h.logger.Info("post_review_comment called",
		zap.String("mr_url", mrURL),
		zap.String("file_path", filePath),
	)

	commentReq := PostCommentRequest{
		FilePath: filePath,
		NewLine:  newLine,
		OldLine:  oldLine,
		Body:     decoratedBody,
	}

	if err := h.postReviewComment(ctx, mrURL, commentReq); err != nil {
		h.logger.Error("post_review_comment failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(`{"status": "comment posted successfully"}`), nil
}

func (h *PostReviewCommentHandler) postReviewComment(ctx context.Context, mrURL string, req PostCommentRequest) error {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := h.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return fmt.Errorf("get MR details: %w", err)
	}

	discussion := gitlabDomain.CreateDiscussionRequest{
		Body: req.Body,
	}

	if req.FilePath != "" {
		discussion.Position = &gitlabDomain.DiscussionPosition{
			BaseSHA:      mr.DiffRefs.BaseSHA,
			StartSHA:     mr.DiffRefs.StartSHA,
			HeadSHA:      mr.DiffRefs.HeadSHA,
			PositionType: "text",
			OldPath:      req.FilePath,
			NewPath:      req.FilePath,
			NewLine:      req.NewLine,
			OldLine:      req.OldLine,
		}
	}

	if err := h.gitlab.PostMRComment(ctx, ref.ProjectPath, ref.MRIID, discussion); err != nil {
		return fmt.Errorf("post review comment: %w", err)
	}

	return nil
}

func intPtrFromFloat(val float64) *int {
	if val == 0 {
		return nil
	}

	intVal := int(val)

	return &intVal
}
