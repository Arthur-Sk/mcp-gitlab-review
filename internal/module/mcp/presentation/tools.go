package presentation

import "github.com/mark3labs/mcp-go/mcp"

func getMRSummaryTool() mcp.Tool {
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

func getChangedFilesTool() mcp.Tool {
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

func getFileContextTool() mcp.Tool {
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

func getArchitectureViewTool() mcp.Tool {
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

func postReviewCommentTool() mcp.Tool {
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
