# MCP GitLab Review

An MCP (Model Context Protocol) server for AI-powered GitLab merge request code reviews. It provides clean, structured, agent-friendly access to GitLab MR data — no raw diffs, no parsing headaches.

## Features

- **Structured diff parsing** — converts raw unified diffs into clean JSON with line numbers, change types, and hunk boundaries
- **File prioritization** — automatically classifies changed files by review importance (high/medium/low/skip), filtering out generated code, vendor files, and lock files
- **Lazy loading** — tools designed for incremental exploration: start with overview, drill into specific files
- **In-memory caching** — aggressive caching of GitLab API responses with configurable TTL
- **Clean architecture** — modular Go design with fx dependency injection, separated by domain concerns

## Tools

| Tool | Description |
|------|-------------|
| `get_mr_summary` | MR metadata: title, description, author, branches, commit messages, status |
| `get_changed_files` | Prioritized list of all changed files with change type, priority, language, line counts |
| `get_file_context` | Full file content at source branch + structured parsed diff for a specific file |
| `get_architecture_view` | Repository folder/file tree at source branch, optionally filtered to subdirectory |
| `post_review_comment` | Post general or line-specific review comments on the MR |

## Architecture

```
cmd/server/              → Entry point
internal/
  app/                   → Application wiring (fx)
  module/
    gitlab/              → GitLab API client, caching service
      domain/            → Domain models, MR URL parser
      repository/        → HTTP client for GitLab REST API
      service/           → Business logic with caching layer
    diff/                → Diff parsing and file classification
      domain/            → Diff models (hunks, lines, priorities)
      service/           → Unified diff parser, file prioritizer
    mcp/                 → MCP server and tool handlers
      presentation/      → Tool definitions, request handlers
      service/           → Orchestration service
  platform/
    config/              → Configuration loading (YAML + env vars)
    logger/              → Zap logger with pretty console output
    cache/               → In-memory TTL cache
config/
  base.yaml              → Base configuration file
```

## Prerequisites

- Go 1.21+
- GitLab personal access token with `api` scope

## Setup

1. Clone the repository:
   ```bash
   git clone <repo-url>
   cd mcp-gitlab-review
   ```

2. Create a `.env` file from the example:
   ```bash
   cp .env.example .env
   ```

3. Edit `.env` and set your GitLab token:
   ```
   GITLAB_PERSONAL_ACCESS_TOKEN=glpat-xxxxxxxxxxxxxxxxxxxx
   GITLAB_API_URL=https://gitlab.com/api/v4
   ```

4. Build:
   ```bash
   go build -o mcp-gitlab-review ./cmd/server
   ```

## Usage

### As an MCP server (stdio transport)

The server communicates via stdin/stdout using the MCP protocol:

```bash
./mcp-gitlab-review
```

### MCP Client Configuration

Add to your MCP client configuration (e.g., Claude Desktop, Cursor, etc.):

```json
{
  "mcpServers": {
    "gitlab-review": {
      "command": "/path/to/mcp-gitlab-review",
      "env": {
        "GITLAB_PERSONAL_ACCESS_TOKEN": "glpat-xxxxxxxxxxxxxxxxxxxx",
        "GITLAB_API_URL": "https://gitlab.com/api/v4"
      }
    }
  }
}
```

### Tool Examples

**Get MR summary:**
```json
{
  "tool": "get_mr_summary",
  "arguments": {
    "mr_url": "https://gitlab.com/group/project/-/merge_requests/123"
  }
}
```

**Get changed files (prioritized):**
```json
{
  "tool": "get_changed_files",
  "arguments": {
    "mr_url": "https://gitlab.com/group/project/-/merge_requests/123"
  }
}
```

**Get file with structured diff:**
```json
{
  "tool": "get_file_context",
  "arguments": {
    "mr_url": "https://gitlab.com/group/project/-/merge_requests/123",
    "file_path": "internal/service/handler.go"
  }
}
```

**Get architecture view:**
```json
{
  "tool": "get_architecture_view",
  "arguments": {
    "mr_url": "https://gitlab.com/group/project/-/merge_requests/123",
    "path": "internal"
  }
}
```

**Post a review comment:**
```json
{
  "tool": "post_review_comment",
  "arguments": {
    "mr_url": "https://gitlab.com/group/project/-/merge_requests/123",
    "body": "This function needs error handling for the nil case.",
    "file_path": "internal/service/handler.go",
    "new_line": 42
  }
}
```

## File Priority Classification

Changed files are automatically classified to help the AI agent focus on what matters:

| Priority | Files |
|----------|-------|
| **High** | New files, business logic, deleted files |
| **Medium** | Test files |
| **Low** | Config files, migrations |
| **Skip** | Generated code (`.pb.go`, `_gen.go`), vendor/, lock files (`go.sum`, `package-lock.json`) |

## Configuration

Configuration is loaded from `config/base.yaml` with environment variable expansion:

```yaml
gitlab:
  apiURL: "${GITLAB_API_URL:https://gitlab.com/api/v4}"
  token: "${GITLAB_PERSONAL_ACCESS_TOKEN}"

cache:
  ttl: 5m
  maxSize: 1000

mcp:
  name: "gitlab-review"
  version: "1.0.0"
```

## Testing

```bash
go test ./... -v
```

## Tech Stack

- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) — MCP protocol implementation
- [go.uber.org/fx](https://github.com/uber-go/fx) — Dependency injection
- [go.uber.org/zap](https://github.com/uber-go/zap) — Structured logging
- [go.uber.org/config](https://github.com/uber-go/config) — YAML configuration
- [joho/godotenv](https://github.com/joho/godotenv) — Environment variable loading
- [thessem/zap-prettyconsole](https://github.com/thessem/zap-prettyconsole) — Pretty console output
- [stretchr/testify](https://github.com/stretchr/testify) — Test assertions

## License

MIT
