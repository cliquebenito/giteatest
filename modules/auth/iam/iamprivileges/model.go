package iampriveleges

import (
	"code.gitea.io/gitea/modules/log"
	"fmt"

	"code.gitea.io/gitea/models/role_model"
)

const sourceControlIAMToolName = "sc"

type WsPrivileges struct {
	// Organization -- org_key в таблице
	Organization string `json:"organization"`

	Roles map[string][]string `json:"rolesMapping"`
}

type RawPrivileges struct {
	WsPrivileges []WsPrivileges `json:"Ws-Privileges"`
}

type Privilege struct {
	//TenantName -- org_key
	TenantName  string `json:"tenant_name"`
	ToolName    string `json:"tool_name"`
	ProjectName string `json:"project_name"`

	role_model.Role `json:"role"`
}

type SourceControlPrivilegesByTenant map[string]Privileges

type Privileges []Privilege

func (p Privileges) GetMaxPrivilege() (Privilege, error) {
	log.Debug(fmt.Sprintf("GetMaxPrivilege %+v", p))
	if p == nil || len(p) == 0 {
		return Privilege{}, fmt.Errorf("empty privelege list")
	}

	if len(p) == 1 {
		return p[0], nil
	}

	maxPrivilege := Privilege{Role: role_model.READER}

	for _, privilege := range p {
		if privilege.Role < maxPrivilege.Role {
			maxPrivilege = privilege
		}
	}

	return maxPrivilege, nil
}
