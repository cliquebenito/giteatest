package response

// ContentsResponse contains information about a repo's entry's (dir, file, symlink, submodule) metadata and content
type ContentsResponse struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	SHA               string `json:"sha"`
	LastCommitSHA     string `json:"last_commit_sha"`
	LastCommitDate    string `json:"last_commit_date"`
	LastCommitMessage string `json:"last_commit_message"`
	// `type` will be `file`, `dir`, `symlink`, or `submodule`
	Type string `json:"type"`
	Size int64  `json:"size"`
	// `encoding` is populated when `type` is `file`, otherwise null
	Encoding *string `json:"encoding"`
	// `content` is populated when `type` is `file`, otherwise null
	Content *string `json:"content"`
	// `target` is populated when `type` is `symlink`, otherwise null
	Target *string `json:"target"`
	// `submodule_git_url` is populated when `type` is `submodule`, otherwise null
	SubmoduleGitURL *string `json:"submodule_git_url"`
	// Язык программирования используемый в файле (если тип файл)
	Language *string `json:"language,omitempty"`
}
