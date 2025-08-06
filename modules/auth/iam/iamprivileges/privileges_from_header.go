package iampriveleges

import (
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"fmt"

	"code.gitea.io/gitea/modules/json"
)

func OpenFromString(content string) (SourceControlPrivilegesByTenant, error) {
	var rawPrivileges RawPrivileges

	jsonContentAsObject := fmt.Sprintf("{\"Ws-Privileges\":%s}", content)
	if err := json.Unmarshal([]byte(jsonContentAsObject), &rawPrivileges); err != nil {
		return nil, fmt.Errorf("unmarshal privileges: %w", err)
	}

	privilegesByOrg := make(SourceControlPrivilegesByTenant)

	// в Ws-Privileges приходит список привилегий для каждой организации и каждого пользователя в OW
	for _, privilege := range rawPrivileges.WsPrivileges {
		for _, value := range privilege.Roles {
			// парсим привилегии из header
			privileges, err := parsePrivileges(value)
			if err != nil {
				return nil, err
			}

			privilegesByOrg[privilege.Organization] = append(privilegesByOrg[privilege.Organization], privileges...)
		}
	}

	return privilegesByOrg, nil
}

func (p SourceControlPrivilegesByTenant) UniqProjectNamesByTenantName(tenantName string) ([]string, error) {
	uniqSourceControlProjectsByTenant := map[string]struct{}{}
	log.Debug(fmt.Sprintf("tenantName: %+v\n", tenantName))
	if tenantName == "" {
		return []string{}, nil
	}
	// в SourceControlPrivilegesByTenant находятся списки привилегий по ключу org_key а в метод поступает имя тенанта из токена
	for _, privilege := range p[tenantName] {
		log.Debug(fmt.Sprintf("setting.SourceControl.IAMToolName: %+v\n", setting.SourceControl.IAMToolName))
		log.Debug(fmt.Sprintf("privilege.TenantName: %+v\n", privilege.TenantName))
		log.Debug(fmt.Sprintf("privilege.ToolName: %+v\n", privilege.ToolName))

		if privilege.TenantName != tenantName ||
			privilege.ToolName != setting.SourceControl.IAMToolName {
			continue
		}

		uniqSourceControlProjectsByTenant[privilege.ProjectName] = struct{}{}
	}

	var sourceControlProjectsByTenant []string
	for projectName := range uniqSourceControlProjectsByTenant {
		sourceControlProjectsByTenant = append(sourceControlProjectsByTenant, projectName)
	}
	log.Debug(fmt.Sprintf("sourceControlProjectsByTenant: %+v\n", sourceControlProjectsByTenant))

	return sourceControlProjectsByTenant, nil
}

func (p SourceControlPrivilegesByTenant) JSON() (string, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("json encoding error: %w", err)
	}

	return string(body), nil
}
