package service

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"mcp-gitlab-review/internal/module/diff/domain"
	gitlabDomain "mcp-gitlab-review/internal/module/gitlab/domain"
)

type Parser struct {
	logger *zap.Logger
}

func NewParser(logger *zap.Logger) *Parser {
	return &Parser{logger: logger}
}

func (p *Parser) Parse(rawDiff string) (*domain.FileDiff, error) {
	if rawDiff == "" {
		return &domain.FileDiff{}, nil
	}

	hunks, err := parseHunks(rawDiff)
	if err != nil {
		return nil, fmt.Errorf("parse hunks: %w", err)
	}

	added, removed := countChanges(hunks)

	return &domain.FileDiff{
		Hunks:        hunks,
		AddedLines:   added,
		RemovedLines: removed,
	}, nil
}

func (p *Parser) ClassifyChange(change gitlabDomain.MergeRequestChange) domain.FileClassification {
	changeType := determineChangeType(change)
	priority, reason := determinePriority(change.NewPath, changeType)
	language := detectLanguage(change.NewPath)

	return domain.FileClassification{
		Path:       change.NewPath,
		ChangeType: changeType,
		Priority:   priority,
		Reason:     reason,
		Language:   language,
	}
}

func parseHunks(rawDiff string) ([]domain.DiffHunk, error) {
	lines := strings.Split(rawDiff, "\n")
	var hunks []domain.DiffHunk
	var currentHunk *domain.DiffHunk
	oldLine, newLine := 0, 0

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}

			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, fmt.Errorf("parse hunk header %q: %w", line, err)
			}

			currentHunk = hunk
			oldLine = hunk.OldStart
			newLine = hunk.NewStart

			continue
		}

		if currentHunk == nil {
			continue
		}

		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}

		diffLine := parseDiffLine(line, &oldLine, &newLine)
		currentHunk.Lines = append(currentHunk.Lines, diffLine)
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks, nil
}

func parseHunkHeader(line string) (*domain.DiffHunk, error) {
	line = strings.TrimPrefix(line, "@@")
	atIdx := strings.Index(line, "@@")
	if atIdx < 0 {
		return nil, fmt.Errorf("invalid hunk header: missing closing @@")
	}

	header := ""
	if atIdx+2 < len(line) {
		header = strings.TrimSpace(line[atIdx+2:])
	}

	rangeStr := strings.TrimSpace(line[:atIdx])
	parts := strings.Split(rangeStr, " ")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid hunk header: expected old and new range")
	}

	oldStart, oldLines, err := parseRange(parts[0])
	if err != nil {
		return nil, fmt.Errorf("parse old range: %w", err)
	}

	newStart, newLines, err := parseRange(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parse new range: %w", err)
	}

	return &domain.DiffHunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
		Header:   header,
	}, nil
}

func parseRange(rangeStr string) (int, int, error) {
	rangeStr = strings.TrimPrefix(rangeStr, "-")
	rangeStr = strings.TrimPrefix(rangeStr, "+")

	parts := strings.Split(rangeStr, ",")

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse start: %w", err)
	}

	lineCount := 1
	if len(parts) > 1 {
		lineCount, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("parse line count: %w", err)
		}
	}

	return start, lineCount, nil
}

func parseDiffLine(line string, oldLine *int, newLine *int) domain.DiffLine {
	if strings.HasPrefix(line, "+") {
		dl := domain.DiffLine{
			Type:      domain.LineAdded,
			Content:   strings.TrimPrefix(line, "+"),
			NewNumber: *newLine,
		}
		*newLine++

		return dl
	}

	if strings.HasPrefix(line, "-") {
		dl := domain.DiffLine{
			Type:      domain.LineRemoved,
			Content:   strings.TrimPrefix(line, "-"),
			OldNumber: *oldLine,
		}
		*oldLine++

		return dl
	}

	content := line
	if strings.HasPrefix(line, " ") {
		content = strings.TrimPrefix(line, " ")
	}

	dl := domain.DiffLine{
		Type:      domain.LineContext,
		Content:   content,
		OldNumber: *oldLine,
		NewNumber: *newLine,
	}
	*oldLine++
	*newLine++

	return dl
}

func countChanges(hunks []domain.DiffHunk) (added int, removed int) {
	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case domain.LineAdded:
				added++
			case domain.LineRemoved:
				removed++
			}
		}
	}

	return added, removed
}

func determineChangeType(change gitlabDomain.MergeRequestChange) domain.FileChangeType {
	switch {
	case change.NewFile:
		return domain.FileNew
	case change.DeletedFile:
		return domain.FileDeleted
	case change.RenamedFile:
		return domain.FileRenamed
	default:
		return domain.FileModified
	}
}

func determinePriority(filePath string, changeType domain.FileChangeType) (domain.FilePriority, string) {
	if isGeneratedFile(filePath) {
		return domain.PrioritySkip, "generated file"
	}

	if isVendorFile(filePath) {
		return domain.PrioritySkip, "vendor/dependency file"
	}

	if isLockFile(filePath) {
		return domain.PrioritySkip, "lock file"
	}

	if changeType == domain.FileNew {
		return domain.PriorityHigh, "new file"
	}

	if isConfigFile(filePath) {
		return domain.PriorityLow, "configuration file"
	}

	if isMigrationFile(filePath) {
		return domain.PriorityLow, "migration file"
	}

	if isTestFile(filePath) {
		return domain.PriorityMedium, "test file"
	}

	return domain.PriorityHigh, "business logic"
}

func isGeneratedFile(filePath string) bool {
	generatedPatterns := []string{
		".pb.go", ".pb.gw.go", "_gen.go", "_generated.go",
		"mock_", "mocks/", ".min.js", ".min.css",
		"swagger.json", "swagger.yaml",
	}

	lower := strings.ToLower(filePath)
	for _, pattern := range generatedPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

func isVendorFile(filePath string) bool {
	vendorPrefixes := []string{"vendor/", "node_modules/", "third_party/"}

	lower := strings.ToLower(filePath)
	for _, prefix := range vendorPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}

func isLockFile(filePath string) bool {
	lockFiles := []string{
		"go.sum", "package-lock.json", "yarn.lock",
		"Gemfile.lock", "poetry.lock", "Cargo.lock",
	}

	base := filepath.Base(filePath)
	for _, lockFile := range lockFiles {
		if base == lockFile {
			return true
		}
	}

	return false
}

func isConfigFile(filePath string) bool {
	configPatterns := []string{
		".yaml", ".yml", ".toml", ".ini", ".conf",
		".env", "Dockerfile", "docker-compose",
		".gitignore", ".editorconfig", "Makefile",
	}

	lower := strings.ToLower(filePath)
	for _, pattern := range configPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

func isMigrationFile(filePath string) bool {
	return strings.Contains(strings.ToLower(filePath), "migration")
}

func isTestFile(filePath string) bool {
	lower := strings.ToLower(filePath)

	return strings.HasSuffix(lower, "_test.go") ||
		strings.Contains(lower, "__tests__") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, "_test.py") ||
		strings.HasSuffix(lower, "spec.rb")
}

func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	languages := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".py":    "Python",
		".rb":    "Ruby",
		".java":  "Java",
		".rs":    "Rust",
		".cpp":   "C++",
		".c":     "C",
		".cs":    "C#",
		".php":   "PHP",
		".kt":    "Kotlin",
		".swift": "Swift",
		".sql":   "SQL",
		".sh":    "Shell",
		".yaml":  "YAML",
		".yml":   "YAML",
		".json":  "JSON",
		".md":    "Markdown",
		".html":  "HTML",
		".css":   "CSS",
	}

	if lang, ok := languages[ext]; ok {
		return lang
	}

	return ""
}
