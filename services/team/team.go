package team

import (
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization/custom"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/forms"
)

// InsertCustomPrivilegeToTeamUser добавляем информацию о добавленных кастомных в команде
func InsertCustomPrivilegeToTeamUser(teamName string, customPrivilegeForm []forms.CustomPrivileges) error {
	if len(customPrivilegeForm) == 0 {
		return fmt.Errorf("incorrect custom privileges request")
	}
	customPrivileges := make([]custom.ScTeamCustomPrivilege, 0, len(customPrivilegeForm))
	for _, customPrivilege := range customPrivilegeForm {
		customPrivileges = append(customPrivileges, custom.ScTeamCustomPrivilege{
			TeamName:         teamName,
			AllRepositories:  customPrivilege.AllRepositories,
			RepositoryID:     customPrivilege.RepoID,
			CustomPrivileges: convertCustomPrivilegeToString(customPrivilege.Privileges),
		})
	}

	dbEngine := db.GetEngine(db.DefaultContext)
	customDb := custom.NewCustomDB(dbEngine)

	if err := customDb.InsertCustomPrivilegesForTeam(customPrivileges); err != nil {
		log.Error("Error has occurred while inserting custom privileges for team: %v", err)
		return fmt.Errorf("adding custom privileges for team: %w", err)
	}
	return nil
}

// convertCustomPrivilegeToString конвертируем массив кастомных привилегий в строку
func convertCustomPrivilegeToString(customPrivileges []role_model.CustomPrivilege) string {
	customToString := make([]string, 0, len(customPrivileges))
	for _, privilege := range customPrivileges {
		customToString = append(customToString, strconv.Itoa(int(privilege)))
	}
	return strings.Join(customToString, ",")
}
