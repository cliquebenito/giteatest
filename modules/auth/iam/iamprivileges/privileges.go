package iampriveleges

import (
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/setting"
	"fmt"
	"strings"
)

func parsePrivileges(rawPrivileges []string) ([]Privilege, error) {
	uniqPrivileges := map[Privilege]struct{}{}

	for _, rawPrivilege := range rawPrivileges {
		privilege, err := parsePrivilege(rawPrivilege)
		if err != nil {
			return nil, fmt.Errorf("parse privilege %s: %w", rawPrivilege, err)
		}
		// поскольку приходит всякий мусор, явно отфильтровываем только адресованную на данный инстанс нагрузку
		if privilege.ToolName != setting.SourceControl.IAMToolName {
			continue
		}

		uniqPrivileges[privilege] = struct{}{}
	}

	var privileges []Privilege
	for privilege := range uniqPrivileges {
		privileges = append(privileges, privilege)
	}

	return privileges, nil
}

func parsePrivilege(rawPrivilege string) (Privilege, error) {
	splittedPrivilege := strings.Split(rawPrivilege, "_")
	if len(splittedPrivilege) != 4 {
		return Privilege{}, fmt.Errorf("invalid privilege format: %s", rawPrivilege)
	}

	rawRole := splittedPrivilege[3]

	var role role_model.Role

	switch rawRole {
	case "x":
		role = role_model.MANAGER
	case "a":
		role = role_model.OWNER
	case "w":
		role = role_model.WRITER
	case "r":
		role = role_model.READER
	default:
		return Privilege{}, fmt.Errorf("invalid role: %s", rawRole)
	}

	// TenantName -- org_key
	privilege := Privilege{
		TenantName:  splittedPrivilege[0],
		ToolName:    splittedPrivilege[1],
		ProjectName: splittedPrivilege[2],
		Role:        role,
	}

	return privilege, nil
}
