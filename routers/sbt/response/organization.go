package response

// Organization организация
type Organization struct {
	ID                        int64  `json:"id"`
	Name                      string `json:"name"`
	FullName                  string `json:"full_name"`
	AvatarURL                 string `json:"avatar_url"`
	Description               string `json:"description"`
	Website                   string `json:"website"`
	Location                  string `json:"location"`
	Visibility                string `json:"visibility"`
	RepoAdminChangeTeamAccess bool   `json:"repo_admin_change_team_access"`
}

// OrganizationListResult список организаций с общим числом организаций
type OrganizationListResult struct {
	Total int64           `json:"total"`
	Data  []*Organization `json:"data"`
}

// OrganizationSettings Настройки организации
type OrganizationSettings struct {
	Name                      string `json:"name"`
	Description               string `json:"description"`
	FullName                  string `json:"full_name"`
	RepoAdminChangeTeamAccess bool   `json:"repo_admin_change_team_access"`
	Location                  string `json:"location"`
	Visibility                string `json:"visibility"`
	Website                   string `json:"website"`
	MaxRepoCreation           int    `json:"max_repo_creation"`
}
