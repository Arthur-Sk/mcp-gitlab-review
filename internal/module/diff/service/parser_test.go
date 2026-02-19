package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/diff/domain"
	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
)

func newTestParser() *Parser {
	logger, _ := zap.NewDevelopment()

	return NewParser(logger)
}

func TestParser_Parse_SingleHunk(t *testing.T) {
	rawDiff := `@@ -10,7 +10,8 @@ func example() {
 context line 1
 context line 2
-removed line
+added line 1
+added line 2
 context line 3
 context line 4
 context line 5`

	parser := newTestParser()
	result, err := parser.Parse(rawDiff)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Hunks, 1)

	hunk := result.Hunks[0]
	assert.Equal(t, 10, hunk.OldStart)
	assert.Equal(t, 7, hunk.OldLines)
	assert.Equal(t, 10, hunk.NewStart)
	assert.Equal(t, 8, hunk.NewLines)
	assert.Equal(t, "func example() {", hunk.Header)

	assert.Equal(t, 2, result.AddedLines)
	assert.Equal(t, 1, result.RemovedLines)

	assert.Equal(t, domain.LineContext, hunk.Lines[0].Type)
	assert.Equal(t, "context line 1", hunk.Lines[0].Content)
	assert.Equal(t, 10, hunk.Lines[0].OldNumber)
	assert.Equal(t, 10, hunk.Lines[0].NewNumber)

	assert.Equal(t, domain.LineRemoved, hunk.Lines[2].Type)
	assert.Equal(t, "removed line", hunk.Lines[2].Content)
	assert.Equal(t, 12, hunk.Lines[2].OldNumber)

	assert.Equal(t, domain.LineAdded, hunk.Lines[3].Type)
	assert.Equal(t, "added line 1", hunk.Lines[3].Content)
	assert.Equal(t, 12, hunk.Lines[3].NewNumber)
}

func TestParser_Parse_MultipleHunks(t *testing.T) {
	rawDiff := `@@ -1,3 +1,4 @@
 line 1
+new line
 line 2
 line 3
@@ -10,3 +11,3 @@ func foo() {
 unchanged
-old
+new
 unchanged`

	parser := newTestParser()
	result, err := parser.Parse(rawDiff)

	require.NoError(t, err)
	require.Len(t, result.Hunks, 2)
	assert.Equal(t, 1, result.Hunks[0].OldStart)
	assert.Equal(t, 10, result.Hunks[1].OldStart)
	assert.Equal(t, 2, result.AddedLines)
	assert.Equal(t, 1, result.RemovedLines)
}

func TestParser_Parse_EmptyDiff(t *testing.T) {
	parser := newTestParser()
	result, err := parser.Parse("")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Hunks)
	assert.Equal(t, 0, result.AddedLines)
	assert.Equal(t, 0, result.RemovedLines)
}

func TestParser_ClassifyChange(t *testing.T) {
	parser := newTestParser()

	tests := []struct {
		name           string
		change         gitlabDomain.MergeRequestChange
		wantChangeType domain.FileChangeType
		wantPriority   domain.FilePriority
	}{
		{
			name: "new Go file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "internal/service/handler.go",
				NewFile: true,
			},
			wantChangeType: domain.FileNew,
			wantPriority:   domain.PriorityHigh,
		},
		{
			name: "modified Go file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "internal/service/handler.go",
			},
			wantChangeType: domain.FileModified,
			wantPriority:   domain.PriorityHigh,
		},
		{
			name: "vendor file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "vendor/github.com/lib/pq/conn.go",
			},
			wantChangeType: domain.FileModified,
			wantPriority:   domain.PrioritySkip,
		},
		{
			name: "generated protobuf file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "api/proto/service.pb.go",
			},
			wantChangeType: domain.FileModified,
			wantPriority:   domain.PrioritySkip,
		},
		{
			name: "test file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "internal/service/handler_test.go",
			},
			wantChangeType: domain.FileModified,
			wantPriority:   domain.PriorityMedium,
		},
		{
			name: "config file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "config/base.yaml",
			},
			wantChangeType: domain.FileModified,
			wantPriority:   domain.PriorityLow,
		},
		{
			name: "lock file",
			change: gitlabDomain.MergeRequestChange{
				NewPath: "go.sum",
			},
			wantChangeType: domain.FileModified,
			wantPriority:   domain.PrioritySkip,
		},
		{
			name: "deleted file",
			change: gitlabDomain.MergeRequestChange{
				NewPath:     "old/deprecated.go",
				DeletedFile: true,
			},
			wantChangeType: domain.FileDeleted,
			wantPriority:   domain.PriorityHigh,
		},
		{
			name: "renamed file",
			change: gitlabDomain.MergeRequestChange{
				OldPath:     "old/name.go",
				NewPath:     "new/name.go",
				RenamedFile: true,
			},
			wantChangeType: domain.FileRenamed,
			wantPriority:   domain.PriorityHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ClassifyChange(tt.change)

			assert.Equal(t, tt.wantChangeType, result.ChangeType)
			assert.Equal(t, tt.wantPriority, result.Priority)
			assert.NotEmpty(t, result.Reason)
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	assert.Equal(t, "Go", detectLanguage("main.go"))
	assert.Equal(t, "Python", detectLanguage("script.py"))
	assert.Equal(t, "JavaScript", detectLanguage("app.js"))
	assert.Equal(t, "TypeScript", detectLanguage("app.ts"))
	assert.Equal(t, "", detectLanguage("unknown.xyz"))
}
