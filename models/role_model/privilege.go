package role_model

import (
	"strconv"

	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

// Privilege тип для описания привилегии получаемой из casbin
type Privilege struct {
	user   string
	tenant string
	org    string
	role   string
}

// EnrichedPrivilege тип для описания дополненной привилегии
type EnrichedPrivilege struct {
	User     *user_model.User
	TenantID string
	Org      *organization.Organization
	Role     Role
}

// convertStringToPrivilegeArray конвертирует массив привилегий из casbin в массив Privilege
func convertStringToPrivilegeArray(privilegeArrayString [][]string) []Privilege {
	privileges := make([]Privilege, 0, len(privilegeArrayString))
	for _, privilegeString := range privilegeArrayString {
		if len(privilegeString) == 4 {
			privileges = append(privileges, Privilege{
				user:   privilegeString[0],
				tenant: privilegeString[1],
				org:    privilegeString[2],
				role:   privilegeString[3],
			})
		}
	}
	return privileges
}

// enrichPrivileges дополняет информацию о привилегиях на основании данных из базы данных
func enrichPrivileges(privileges []Privilege) ([]EnrichedPrivilege, error) {
	userIds := make([]int64, 0)
	enrichedPrivileges := make([]EnrichedPrivilege, 0)
	for _, privilege := range privileges {
		userId, err := strconv.ParseInt(privilege.user, 10, 0)
		if err != nil {
			log.Error("Error has occurred while parsing userId: %s. Error: %v", privilege.user, err)
			return nil, err
		}
		userIds = append(userIds, userId)
		orgId, err := strconv.ParseInt(privilege.org, 10, 0)
		if err != nil {
			log.Error("Error has occurred while parsing orgId: %s. Error: %v", privilege.org, err)
			return nil, err
		}
		userIds = append(userIds, orgId)
	}

	users, err := user_model.GetUsersByIDs(userIds)
	if err != nil {
		log.Error("Error has occurred while getting users by ids. Error: %v", err)
		return nil, err
	}

	userMap := users.GetUserMap()

	for _, privilege := range privileges {
		userId, err := strconv.ParseInt(privilege.user, 10, 0)
		if err != nil {
			log.Error("Error has occurred while parsing userId: %s. Error: %v", privilege.user, err)
			return nil, err
		}
		orgId, err := strconv.ParseInt(privilege.org, 10, 0)
		if err != nil {
			log.Error("Error has occurred while parsing orgId: %s. Error: %v", privilege.org, err)
			return nil, err
		}
		role, ok := GetRoleByString(privilege.role)
		if !ok {
			return nil, &ErrNonExistentRole{Role: privilege.role}
		}

		if userMap[userId] != nil && userMap[orgId] != nil {
			enrichedPrivileges = append(enrichedPrivileges, EnrichedPrivilege{
				User:     userMap[userId],
				TenantID: privilege.tenant,
				Org:      organization.OrgFromUser(userMap[orgId]),
				Role:     role,
			})
		}
	}

	return enrichedPrivileges, nil
}
