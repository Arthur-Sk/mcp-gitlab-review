package tools

import (
	"context"
	"fmt"
	"mcp-gitlab-review/internal/module/mcp/service/helper"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	diffDomain "mcp-gitlab-review/internal/module/diff/domain"
	diffService "mcp-gitlab-review/internal/module/diff/service"
	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
	gitlabService "mcp-gitlab-review/internal/module/gitlab/service"
)

type FileContextResponse struct {
	Path        string               `json:"path"`
	Language    string               `json:"language,omitempty"`
	FileContent string               `json:"file_content"`
	Diff        *diffDomain.FileDiff `json:"diff"`
}

type GetFileContextHandler struct {
	gitlab     *gitlabService.Service
	diffParser *diffService.Parser
	logger     *zap.Logger
}

func NewGetFileContextHandler(gitlab *gitlabService.Service, diffParser *diffService.Parser, logger *zap.Logger) *GetFileContextHandler {
	return &GetFileContextHandler{
		gitlab:     gitlab,
		diffParser: diffParser,
		logger:     logger,
	}
}

func (h *GetFileContextHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"get_file_context",
		mcp.WithDescription(
			"Get the full file content at the source branch plus the structured, "+
				"parsed diff for a specific file. Returns clean hunks with line numbers, "+
				"change types (added/removed/context), and the complete file for reference. "+
				"Call this per-file for detailed code review.",
		),
		mcp.WithString(
			"mr_url",
			mcp.Required(),
			mcp.Description("Full GitLab merge request URL"),
		),
		mcp.WithString(
			"file_path",
			mcp.Required(),
			mcp.Description("Path of the file to inspect (as returned by get_changed_files)"),
		),
	)
}

func (h *GetFileContextHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	result, err := h.getFileContext(ctx, mrURL, filePath)
	if err != nil {
		h.logger.Error("get_file_context failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *GetFileContextHandler) getFileContext(ctx context.Context, mrURL string, filePath string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := h.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR details: %w", err)
	}

	changes, err := h.gitlab.GetMRChanges(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR changes: %w", err)
	}

	var matchedChange *gitlabDomain.MergeRequestChange
	for i, change := range changes.Changes {
		if change.NewPath == filePath || change.OldPath == filePath {
			matchedChange = &changes.Changes[i]

			break
		}
	}

	if matchedChange == nil {
		return "", fmt.Errorf("file %q not found in MR changes", filePath)
	}

	diff, err := h.diffParser.Parse(matchedChange.Diff)
	if err != nil {
		return "", fmt.Errorf("parse diff for %q: %w", filePath, err)
	}

	diff.OldPath = matchedChange.OldPath
	diff.NewPath = matchedChange.NewPath
	diff.NewFile = matchedChange.NewFile
	diff.DeletedFile = matchedChange.DeletedFile
	diff.RenamedFile = matchedChange.RenamedFile

	var fileContent string
	if !matchedChange.DeletedFile {
		fileContent, err = h.gitlab.GetFileContent(ctx, ref.ProjectPath, mr.SourceBranch, filePath)
		if err != nil {
			h.logger.Warn("failed to fetch file content",
				zap.String("file", filePath),
				zap.Error(err),
			)
		}
	}

	classification := h.diffParser.ClassifyChange(*matchedChange)

	response := FileContextResponse{
		Path:        filePath,
		Language:    classification.Language,
		FileContent: fileContent,
		Diff:        diff,
	}

	return helper.ToJSON(response)
}
