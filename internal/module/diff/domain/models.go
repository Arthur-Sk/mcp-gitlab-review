package domain

type LineType string

const (
	LineContext LineType = "context"
	LineAdded   LineType = "added"
	LineRemoved LineType = "removed"
)

type DiffLine struct {
	Type      LineType `json:"type"`
	Content   string   `json:"content"`
	OldNumber int      `json:"old_number,omitempty"`
	NewNumber int      `json:"new_number,omitempty"`
}

type DiffHunk struct {
	OldStart int        `json:"old_start"`
	OldLines int        `json:"old_lines"`
	NewStart int        `json:"new_start"`
	NewLines int        `json:"new_lines"`
	Header   string     `json:"header,omitempty"`
	Lines    []DiffLine `json:"lines"`
}

type FileDiff struct {
	OldPath      string     `json:"old_path"`
	NewPath      string     `json:"new_path"`
	NewFile      bool       `json:"new_file"`
	DeletedFile  bool       `json:"deleted_file"`
	RenamedFile  bool       `json:"renamed_file"`
	Hunks        []DiffHunk `json:"hunks"`
	AddedLines   int        `json:"added_lines"`
	RemovedLines int        `json:"removed_lines"`
}

type FileChangeType string

const (
	FileNew      FileChangeType = "new"
	FileModified FileChangeType = "modified"
	FileDeleted  FileChangeType = "deleted"
	FileRenamed  FileChangeType = "renamed"
)

type FilePriority string

const (
	PriorityHigh   FilePriority = "high"
	PriorityMedium FilePriority = "medium"
	PriorityLow    FilePriority = "low"
	PrioritySkip   FilePriority = "skip"
)

type FileClassification struct {
	Path       string         `json:"path"`
	ChangeType FileChangeType `json:"change_type"`
	Priority   FilePriority   `json:"priority"`
	Reason     string         `json:"reason"`
	Language   string         `json:"language,omitempty"`
}
