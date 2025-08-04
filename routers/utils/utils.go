// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"context"
	"html"
	"net/url"
	"strings"
)

// RemoveUsernameParameterSuffix returns the username parameter without the (fullname) suffix - leaving just the username
func RemoveUsernameParameterSuffix(name string) string {
	if index := strings.Index(name, " ("); index >= 0 {
		name = name[:index]
	}
	return name
}

// SanitizeFlashErrorString will sanitize a flash error string
func SanitizeFlashErrorString(x string) string {
	return strings.ReplaceAll(html.EscapeString(x), "\n", "<br>")
}

// IsExternalURL checks if rawURL points to an external URL like http://example.com
func IsExternalURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	appURL, _ := url.Parse(setting.AppURL)
	if len(parsed.Host) != 0 && strings.Replace(parsed.Host, "www.", "", 1) != strings.Replace(appURL.Host, "www.", "", 1) {
		return true
	}
	return false
}

// GetTenantsPrivilegesByUserID получаем привилегии для тенанта по userID
func GetTenantsPrivilegesByUserID(ctx context.Context, userID int64) ([]role_model.EnrichedPrivilege, error) {
	tenantIDs, err := role_model.GetUserTenantIDsOrDefaultTenantID(&user_model.User{ID: userID})
	if err != nil {
		log.Error("GetTenantsPrivilegesByUserID. GetUserTenantIds: err %v", err)
		return nil, err
	}
	var allPrivilege []role_model.EnrichedPrivilege
	for _, tenantID := range tenantIDs {
		tenantPrivileges, err := role_model.GetPrivilegesByTenant(tenantID)
		if err != nil {
			log.Error("GetTenantsPrivilegesByUserID. GetPrivilegesByTenant: err %v", err)
			return nil, err
		}
		allPrivilege = append(allPrivilege, tenantPrivileges...)
	}
	return allPrivilege, nil
}

// ConvertPrivilegesTenantFromOrganizationsOrUsers конвектирую массив привилегий тенанта в мапу id user или organization
func ConvertPrivilegesTenantFromOrganizationsOrUsers(tenantPrivileges []role_model.EnrichedPrivilege, typeUser user_model.UserType) map[int64]struct{} {
	tenantIDs := make(map[string]struct{})
	for _, ten := range tenantPrivileges {
		tenantIDs[ten.TenantID] = struct{}{}
	}
	tenantEntities := make(map[string]bool)
	for ten := range tenantIDs {
		tenantEntity, err := tenant.GetTenantByID(db.DefaultContext, ten)
		if err != nil {
			log.Error("ConvertPrivilegesTenantFromOrganizationsOrUsers tenant.GetTenantByI failed: %v", err)
			return nil
		}
		tenantEntities[ten] = tenantEntity.IsActive
	}
	privileges := make(map[int64]struct{})
	switch typeUser {
	case user_model.UserTypeIndividual:
		for _, tenantPrivilege := range tenantPrivileges {
			if tenantEntities[tenantPrivilege.TenantID] {
				privileges[tenantPrivilege.User.ID] = struct{}{}
			}
		}
	default:
		for _, tenantPrivilege := range tenantPrivileges {
			if tenantEntities[tenantPrivilege.TenantID] {
				privileges[tenantPrivilege.Org.ID] = struct{}{}
			}
		}
	}
	return privileges
}

// ConvertMapUserOrOrganizationsInSlice мапу в массив id user или organization
func ConvertMapUserOrOrganizationsInSlice(userIDs map[int64]struct{}) []int64 {
	userOrOrganizationIDs := make([]int64, 0, len(userIDs))
	for userOrOrganization := range userIDs {
		userOrOrganizationIDs = append(userOrOrganizationIDs, userOrOrganization)
	}
	return userOrOrganizationIDs
}

// ConvertTenantPrivilegesInOrganizations конвертируем массив привилегий для тенатнов в массив организаций
func ConvertTenantPrivilegesInOrganizations(tenantPrivileges []role_model.EnrichedPrivilege) []*organization.Organization {
	tenantIDs := make(map[string]struct{})
	for _, ten := range tenantPrivileges {
		tenantIDs[ten.TenantID] = struct{}{}
	}
	tenantEntities := make(map[string]bool)
	for ten := range tenantIDs {
		tenantEntity, err := tenant.GetTenantByID(db.DefaultContext, ten)
		if err != nil {
			log.Error("ConvertPrivilegesTenantFromOrganizationsOrUsers tenant.GetTenantByI failed: %v", err)
			return nil
		}
		tenantEntities[ten] = tenantEntity.IsActive
	}
	organizations := make([]*organization.Organization, 0, len(tenantPrivileges))
	organizationPrivileges := make(map[*organization.Organization]struct{})
	for _, tenantPrivilege := range tenantPrivileges {
		if tenantEntities[tenantPrivilege.TenantID] && tenantPrivilege.Org.Type == 1 {
			organizationPrivileges[tenantPrivilege.Org] = struct{}{}
		}
	}
	for organizationPrivilege := range organizationPrivileges {
		organizations = append(organizations, organizationPrivilege)
	}
	return organizations
}
