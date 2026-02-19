package domain

type Author struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type DiffRefs struct {
	BaseSHA  string `json:"base_sha"`
	HeadSHA  string `json:"head_sha"`
	StartSHA string `json:"start_sha"`
}

type MergeRequest struct {
	IID            int      `json:"iid"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	State          string   `json:"state"`
	SourceBranch   string   `json:"source_branch"`
	TargetBranch   string   `json:"target_branch"`
	Author         Author   `json:"author"`
	WebURL         string   `json:"web_url"`
	DiffRefs       DiffRefs `json:"diff_refs"`
	SHA            string   `json:"sha"`
	HasConflicts   bool     `json:"has_conflicts"`
	ChangesCount   string   `json:"changes_count"`
	MergeStatus    string   `json:"merge_status"`
	CommitMessages []string `json:"-"`
}

type MergeRequestChange struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	Diff        string `json:"diff"`
}

type MergeRequestChanges struct {
	MergeRequest
	Changes []MergeRequestChange `json:"changes"`
}

type Commit struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type TreeEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type DiscussionPosition struct {
	BaseSHA      string `json:"base_sha"`
	StartSHA     string `json:"start_sha"`
	HeadSHA      string `json:"head_sha"`
	PositionType string `json:"position_type"`
	OldPath      string `json:"old_path"`
	NewPath      string `json:"new_path"`
	OldLine      *int   `json:"old_line,omitempty"`
	NewLine      *int   `json:"new_line,omitempty"`
}

type CreateDiscussionRequest struct {
	Body     string              `json:"body"`
	Position *DiscussionPosition `json:"position,omitempty"`
}
