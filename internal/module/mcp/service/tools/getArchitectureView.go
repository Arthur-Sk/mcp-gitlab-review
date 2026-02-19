package tools

import (
	"context"
	"fmt"
	"mcp-gitlab-review/internal/module/mcp/service/helper"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
	gitlabService "mcp-gitlab-review/internal/module/gitlab/service"
)

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

type GetArchitectureViewHandler struct {
	gitlab *gitlabService.Service
	logger *zap.Logger
}

func NewGetArchitectureViewHandler(gitlab *gitlabService.Service, logger *zap.Logger) *GetArchitectureViewHandler {
	return &GetArchitectureViewHandler{
		gitlab: gitlab,
		logger: logger,
	}
}

func (h *GetArchitectureViewHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"get_architecture_view",
		mcp.WithDescription(
			"Get the full repository folder/file tree at the source branch. "+
				"Use this to understand the project structure and architecture "+
				"before or during code review. Optionally filter to a subdirectory.",
		),
		mcp.WithString(
			"mr_url",
			mcp.Required(),
			mcp.Description("Full GitLab merge request URL"),
		),
		mcp.WithString(
			"path",
			mcp.Description("Optional subdirectory path to filter the tree"),
		),
	)
}

func (h *GetArchitectureViewHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mrURL, err := request.RequireString("mr_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	path := request.GetString("path", "")

	h.logger.Info("get_architecture_view called",
		zap.String("mr_url", mrURL),
		zap.String("path", path),
	)

	result, err := h.getArchitectureView(ctx, mrURL, path)
	if err != nil {
		h.logger.Error("get_architecture_view failed", zap.Error(err))

		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (h *GetArchitectureViewHandler) getArchitectureView(ctx context.Context, mrURL string, path string) (string, error) {
	ref, err := gitlabDomain.ParseMRURL(mrURL)
	if err != nil {
		return "", fmt.Errorf("parse MR URL: %w", err)
	}

	mr, err := h.gitlab.GetMRSummary(ctx, ref.ProjectPath, ref.MRIID)
	if err != nil {
		return "", fmt.Errorf("get MR details: %w", err)
	}

	entries, err := h.gitlab.GetRepositoryTree(ctx, ref.ProjectPath, mr.SourceBranch, path)
	if err != nil {
		return "", fmt.Errorf("get repository tree: %w", err)
	}

	tree := buildTree(entries)

	response := ArchitectureViewResponse{
		Branch: mr.SourceBranch,
		Tree:   tree,
	}

	return helper.ToJSON(response)
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
