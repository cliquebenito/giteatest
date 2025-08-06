package response

import "time"

// Permission доступ
type Permission struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

// InternalTracker настройки внутреннего трекера
type InternalTracker struct {
	EnableTimeTracker                bool `json:"enable_time_tracker"`
	AllowOnlyContributorsToTrackTime bool `json:"allow_only_contributors_to_track_time"`
	EnableIssueDependencies          bool `json:"enable_issue_dependencies"`
}

// ExternalTracker нестройки внешнего трекера
type ExternalTracker struct {
	// Адрес внешнего трекера
	ExternalTrackerURL string `json:"external_tracker_url"`
	// Формат адреса. Должны быть использованы {user}, {repo} and {index} для имени пользователя, имени репозитория и номера задачи( пулл реквеста)
	ExternalTrackerFormat string `json:"external_tracker_format"`
	// External Формат номера задачи - `numeric`, `alphanumeric`, или `regexp`
	ExternalTrackerStyle string `json:"external_tracker_style"`
	// External Паттерн задачи
	ExternalTrackerRegexpPattern string `json:"external_tracker_regexp_pattern"`
}

// ExternalWiki настройки внешней вики
type ExternalWiki struct {
	ExternalWikiURL string `json:"external_wiki_url"`
}

// Repository репозиторий
type Repository struct {
	ID                            int64            `json:"id"`
	Owner                         *User            `json:"owner"`
	Name                          string           `json:"name"`
	FullName                      string           `json:"full_name"`
	Description                   string           `json:"description"`
	Empty                         bool             `json:"empty"`
	Private                       bool             `json:"private"`
	Fork                          bool             `json:"fork"`
	Template                      bool             `json:"template"`
	Parent                        *Repository      `json:"parent"`
	Mirror                        bool             `json:"mirror"`
	Size                          int              `json:"size"`
	Language                      string           `json:"language"`
	Link                          string           `json:"link"`
	SSHURL                        string           `json:"ssh_url"`
	CloneURL                      string           `json:"clone_url"`
	Website                       string           `json:"website"`
	Stars                         int              `json:"stars_count"`
	Forks                         int              `json:"forks_count"`
	Watchers                      int              `json:"watchers_count"`
	OpenIssues                    int              `json:"open_issues_count"`
	OpenPulls                     int              `json:"open_pr_counter"`
	Releases                      int              `json:"release_counter"`
	DefaultBranch                 string           `json:"default_branch"`
	Archived                      bool             `json:"archived"`
	Created                       time.Time        `json:"created_at"`
	Updated                       time.Time        `json:"updated_at"`
	ArchivedAt                    time.Time        `json:"archived_at"`
	Permissions                   *Permission      `json:"permissions,omitempty"`
	HasIssues                     bool             `json:"has_issues"`
	InternalTracker               *InternalTracker `json:"internal_tracker,omitempty"`
	ExternalTracker               *ExternalTracker `json:"external_tracker,omitempty"`
	HasWiki                       bool             `json:"has_wiki"`
	ExternalWiki                  *ExternalWiki    `json:"external_wiki,omitempty"`
	HasPullRequests               bool             `json:"has_pull_requests"`
	HasProjects                   bool             `json:"has_projects"`
	HasReleases                   bool             `json:"has_releases"`
	HasPackages                   bool             `json:"has_packages"`
	HasActions                    bool             `json:"has_actions"`
	IgnoreWhitespaceConflicts     bool             `json:"ignore_whitespace_conflicts"`
	AllowMerge                    bool             `json:"allow_merge_commits"`
	AllowRebase                   bool             `json:"allow_rebase"`
	AllowRebaseMerge              bool             `json:"allow_rebase_explicit"`
	AllowSquash                   bool             `json:"allow_squash_merge"`
	AllowRebaseUpdate             bool             `json:"allow_rebase_update"`
	DefaultDeleteBranchAfterMerge bool             `json:"default_delete_branch_after_merge"`
	DefaultMergeStyle             string           `json:"default_merge_style"`
	DefaultAllowMaintainerEdit    bool             `json:"default_allow_maintainer_edit"`
	AvatarURL                     string           `json:"avatar_url"`
	Internal                      bool             `json:"internal"`
	MirrorInterval                string           `json:"mirror_interval"`
	MirrorUpdated                 time.Time        `json:"mirror_updated,omitempty"`
	RepoTransfer                  *RepoTransfer    `json:"repo_transfer"`
	IsWatching                    bool             `json:"is_watching"`
	IsStarring                    bool             `json:"is_starring"`
}

// RepoTransfer настройки трансфера репозитория
type RepoTransfer struct {
	Doer      *User   `json:"doer"`
	Recipient *User   `json:"recipient"`
	Teams     []*Team `json:"teams"`
}
