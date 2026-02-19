package tools

import (
	"context"
	"fmt"
	"mcp-gitlab-review/internal/module/mcp/service/helper"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	diffDomain "mcp-gitlab-review/internal/module/diff/domain"
	diffService "mcp-gitlab-review/internal/module/diff/service"
	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
	gitlabService "mcp-gitlab-review/internal/module/gitlab/service"
)

type ChangedFileEntry struct {
	Path         string                    `json:"path"`
	ChangeType   diffDomain.FileChangeType `json:"change_type"`
	Priority     diffDomain.FilePriority   `json:"priority"`
	Reason       string                    `json:"reason"`
	Language     string                    `json:"language,omitempty"`
	AddedLines   int                       `json:"added_lines"`
	RemovedLines int                       `json:"removed_lines"`
	OldPath      string                    `json:"old_path,omitempty"`
}

type ChangedFilesResponse struct {
	TotalFiles int                `json:"total_files"`
	Summary    ChangesSummary     `json:"summary"`
	Files      []ChangedFileEntry `json:"files"`
}

type ChangesSummary struct {
	NewFiles      int `json:"new_files"`
	ModifiedFiles int `json:"modified_files"`
	DeletedFiles  int `json:"deleted_files"`
	RenamedFiles  int `json:"renamed_files"`
	TotalAdded    int `json:"total_added_lines"`
	TotalRemoved  int `json:"total_removed_lines"`
}

type GetChangedFilesHandler struct {
	gitlab     *gitlabService.Service
	diffParser *diffService.Parser
	logger     *zap.Logger
}

func NewGetChangedFilesHandler(gitlab *gitlabService.Service, diffParser *diffService.Parser, logger *zap.Logger) *GetChangedFilesHandler {
	return &GetChangedFilesHandler{
		gitlab:     gitlab,
		diffParser: diffParser,
		logger:     logger,
	}
}

func (h *GetChangedFilesHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"get_changed_files",
		mcp.WithDescription(
			"Get a prioritized list of all changed files in the MR with metadata: "+
				"change type (new/modified/deleted/renamed), priority classification "+
				"(high/medium/low/skip), language, and line change counts. "+
				"Files are sorted by review priority. Use this to plan which files to review in detail.",
		),
		mcp.WithString(
			"mr_url",
			mcp.Required(),
			mcp.Description("Full GitLab merge request URL"),
		),
	)
}

func (h *GetChangedFilesHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.logger.Info("get_changed_files called", zap.String("mr_url", mrURL))

	result, err := h.getChangedFiles(ctx, mrURL)
	if err != nil {
		h.logger.Error("get_changed_files failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *GetChangedFilesHandler) getChangedFiles(ctx context.Context, mrURL string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	changes, err := h.gitlab.GetMRChanges(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR changes: %w", err)
	}

	var files []ChangedFileEntry
	var summary ChangesSummary

	for _, change := range changes.Changes {
		classification := h.diffParser.ClassifyChange(change)

		diff, err := h.diffParser.Parse(change.Diff)
		if err != nil {
			h.logger.Warn("failed to parse diff, skipping line counts",
				zap.String("file", change.NewPath),
				zap.Error(err),
			)
		}

		entry := ChangedFileEntry{
			Path:       change.NewPath,
			ChangeType: classification.ChangeType,
			Priority:   classification.Priority,
			Reason:     classification.Reason,
			Language:   classification.Language,
		}

		if diff != nil {
			entry.AddedLines = diff.AddedLines
			entry.RemovedLines = diff.RemovedLines
			summary.TotalAdded += diff.AddedLines
			summary.TotalRemoved += diff.RemovedLines
		}

		if change.RenamedFile && change.OldPath != change.NewPath {
			entry.OldPath = change.OldPath
		}

		switch classification.ChangeType {
		case diffDomain.FileNew:
			summary.NewFiles++
		case diffDomain.FileModified:
			summary.ModifiedFiles++
		case diffDomain.FileDeleted:
			summary.DeletedFiles++
		case diffDomain.FileRenamed:
			summary.RenamedFiles++
		}

		files = append(files, entry)
	}

	sortFilesByPriority(files)

	response := ChangedFilesResponse{
		TotalFiles: len(files),
		Summary:    summary,
		Files:      files,
	}

	return helper.ToJSON(response)
}

func sortFilesByPriority(files []ChangedFileEntry) {
	priorityOrder := map[diffDomain.FilePriority]int{
		diffDomain.PriorityHigh:   0,
		diffDomain.PriorityMedium: 1,
		diffDomain.PriorityLow:    2,
		diffDomain.PrioritySkip:   3,
	}

	sort.Slice(files, func(i, j int) bool {
		pi := priorityOrder[files[i].Priority]
		pj := priorityOrder[files[j].Priority]

		if pi != pj {
			return pi < pj
		}

		return files[i].Path < files[j].Path
	})
}
