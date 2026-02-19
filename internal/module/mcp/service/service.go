package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"

	diffDomain "mcp-gitlab-review/internal/module/diff/domain"
	diffService "mcp-gitlab-review/internal/module/diff/service"
	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
	gitlabService "mcp-gitlab-review/internal/module/gitlab/service"
)

type Service struct {
	gitlab     *gitlabService.Service
	diffParser *diffService.Parser
	logger     *zap.Logger
}

func NewService(gitlab *gitlabService.Service, diffParser *diffService.Parser, logger *zap.Logger) *Service {
	return &Service{
		gitlab:     gitlab,
		diffParser: diffParser,
		logger:     logger,
	}
}

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

func (s *Service) GetMRSummary(ctx context.Context, mrURL string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := s.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
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

	return toJSON(response)
}

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

func (s *Service) GetChangedFiles(ctx context.Context, mrURL string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	changes, err := s.gitlab.GetMRChanges(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR changes: %w", err)
	}

	var files []ChangedFileEntry
	var summary ChangesSummary

	for _, change := range changes.Changes {
		classification := s.diffParser.ClassifyChange(change)

		diff, err := s.diffParser.Parse(change.Diff)
		if err != nil {
			s.logger.Warn("failed to parse diff, skipping line counts",
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

	return toJSON(response)
}

type FileContextResponse struct {
	Path        string               `json:"path"`
	Language    string               `json:"language,omitempty"`
	FileContent string               `json:"file_content"`
	Diff        *diffDomain.FileDiff `json:"diff"`
}

func (s *Service) GetFileContext(ctx context.Context, mrURL string, filePath string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := s.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR details: %w", err)
	}

	changes, err := s.gitlab.GetMRChanges(ctx, ref.ProjectPath, ref.MRIID)
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

	diff, err := s.diffParser.Parse(matchedChange.Diff)
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
		fileContent, err = s.gitlab.GetFileContent(ctx, ref.ProjectPath, mr.SourceBranch, filePath)
		if err != nil {
			s.logger.Warn("failed to fetch file content",
				zap.String("file", filePath),
				zap.Error(err),
			)
		}
	}

	classification := s.diffParser.ClassifyChange(*matchedChange)

	response := FileContextResponse{
		Path:        filePath,
		Language:    classification.Language,
		FileContent: fileContent,
		Diff:        diff,
	}

	return toJSON(response)
}

type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"`
	Children []TreeNode `json:"children,omitempty"`
}

type ArchitectureViewResponse struct {
	Branch string     `json:"branch"`
	Tree   []TreeNode `json:"tree"`
}

func (s *Service) GetArchitectureView(ctx context.Context, mrURL string, path string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := s.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR details: %w", err)
	}

	entries, err := s.gitlab.GetRepositoryTree(ctx, ref.ProjectPath, mr.SourceBranch, path)
	if err != nil {
		return "", fmt.Errorf("get repository tree: %w", err)
	}

	tree := buildTree(entries)

	response := ArchitectureViewResponse{
		Branch: mr.SourceBranch,
		Tree:   tree,
	}

	return toJSON(response)
}

type PostCommentRequest struct {
	FilePath string `json:"file_path"`
	NewLine  *int   `json:"new_line,omitempty"`
	OldLine  *int   `json:"old_line,omitempty"`
	Body     string `json:"body"`
}

func (s *Service) PostReviewComment(ctx context.Context, mrURL string, req PostCommentRequest) error {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := s.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
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

	if err := s.gitlab.PostMRComment(ctx, ref.ProjectPath, ref.MRIID, discussion); err != nil {
		return fmt.Errorf("post review comment: %w", err)
	}

	return nil
}

func buildTree(entries []gitlabDomain.TreeEntry) []TreeNode {
	nodeMap := make(map[string]*TreeNode)
	var roots []TreeNode

	for _, e := range entries {
		node := TreeNode{
			Name: e.Name,
			Path: e.Path,
			Type: e.Type,
		}

		if e.Type == "tree" {
			node.Children = []TreeNode{}
		}

		nodeMap[e.Path] = &node
	}

	for _, e := range entries {
		parentPath := parentDir(e.Path)
		if parent, ok := nodeMap[parentPath]; ok {
			parent.Children = append(parent.Children, *nodeMap[e.Path])
		} else {
			roots = append(roots, *nodeMap[e.Path])
		}
	}

	return roots
}

func parentDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}

	return path[:idx]
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

func toJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}

	return string(data), nil
}
