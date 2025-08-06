package response

// Team команда (группа пользователей) в организации
type Team struct {
	ID                      int64         `json:"id"`
	Name                    string        `json:"name"`
	Description             string        `json:"description"`
	Organization            *Organization `json:"organization"`
	IncludesAllRepositories bool          `json:"includes_all_repositories"`
	// enum: none,read,write,admin,owner
	Permission string `json:"permission"`
	// пример: {"repo.code":"read","repo.issues":"write","repo.ext_issues":"none","repo.wiki":"admin","repo.pulls":"owner","repo.releases":"none","repo.projects":"none","repo.ext_wiki":"none"}
	UnitsMap         map[string]string `json:"units_map"`
	CanCreateOrgRepo bool              `json:"can_create_org_repo"`
}
