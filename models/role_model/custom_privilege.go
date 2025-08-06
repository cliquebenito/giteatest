package role_model

import (
	"sort"
	"strings"

	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

// CustomPrivilege тип для перечисления привилегий
type CustomPrivilege int

// ConfCustomPrivilege структура для проверки доступа пользователя
type ConfCustomPrivilege struct {
	TenantID        string
	User            *user_model.User
	Org             *organization.Organization
	Action          Action
	RepoID          int64
	CustomPrivilege CustomPrivilege
}

// Перечисляем все привилегии
const (
	ViewBranch CustomPrivilege = iota + 1
	ChangeBranch
	CreatePR
	ApprovePR
	MergePR
)

// Перечисляем индексы для групповой политики g3
const (
	TeamName int = iota
	UserID
	RepositoryID
)

// описание привилегий
var (
	Privileges = map[CustomPrivilege]string{
		ViewBranch:   "viewBranch",
		ChangeBranch: "changeBranch",
		CreatePR:     "createPR",
		ApprovePR:    "approvePR",
		MergePR:      "mergePR",
	}
	// Сокращенное название для привилегий
	namesOfPolicy = map[CustomPrivilege]string{
		ViewBranch:   "vB",
		ChangeBranch: "chB",
		CreatePR:     "cPR",
		ApprovePR:    "aPr",
		MergePR:      "mPr",
	}
	PolicyOfNames = map[string]CustomPrivilege{
		"vB":  ViewBranch,
		"chB": ChangeBranch,
		"cPR": CreatePR,
		"aPr": ApprovePR,
		"mPr": MergePR,
	}
)

// String возвращаем описание привилегий
func (p CustomPrivilege) String() string {
	return Privileges[p]
}

// GetCustomPrivilegesByString метод для получения CustomPrivilege по строке
func GetCustomPrivilegesByString(str string) (CustomPrivilege, bool) {
	for privilege, privilegeString := range Privileges {
		if privilegeString == str {
			return privilege, true
		}
	}
	return 0, false
}

// ConvertCustomPrivilegeToNameOfPolicy конвертируем массив кастомных привилегий в имя политики
func ConvertCustomPrivilegeToNameOfPolicy(customPrivileges []CustomPrivilege) string {
	if len(customPrivileges) == 0 {
		log.Debug("incorrect length custom privileges array")
		return ""
	}

	sort.Slice(customPrivileges, func(i, j int) bool {
		return customPrivileges[i] < customPrivileges[j]
	})

	policy := strings.Builder{}
	for _, privilege := range customPrivileges {
		if _, err := policy.WriteString(namesOfPolicy[privilege] + "_"); err != nil {
			log.Error("Error has occurred while trying to write string: %v", err)
			return ""
		}
	}

	if len(policy.String()) == 0 {
		log.Debug("incorrect length policy")
		return ""
	}
	return policy.String()[0 : len(policy.String())-1]
}

// ConvertConflictPolices конвертирует конфликтующие привилегии из старых и новых привилегий
func ConvertConflictPolices(old, new string) string {
	oldCustomPrivileges := strings.Split(old, "_")
	newCustomPrivileges := strings.Split(new, "_")

	uniqueCustomPrivileges := make(map[string]struct{}, len(oldCustomPrivileges))
	for idx := range oldCustomPrivileges {
		uniqueCustomPrivileges[oldCustomPrivileges[idx]] = struct{}{}
	}

	for idx := range newCustomPrivileges {
		uniqueCustomPrivileges[newCustomPrivileges[idx]] = struct{}{}
	}

	newCustom := make([]CustomPrivilege, 0, len(uniqueCustomPrivileges))
	for privilege := range uniqueCustomPrivileges {
		newCustom = append(newCustom, PolicyOfNames[privilege])
	}
	return ConvertCustomPrivilegeToNameOfPolicy(newCustom)
}
