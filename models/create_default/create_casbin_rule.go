package create_default

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
)

// CreateCasbinRule метод для создания ролей casbin rule
func CreateCasbinRule(ctx context.Context) error {

	if !setting.SourceControl.TenantWithRoleModeEnabled {
		return nil
	}

	err := FindSequenses(ctx)
	if err != nil {
		return err
	}

	// todo если тенантов будет много, стоит задуматься о присвоении прав на просмотр для нескольких тенантов.
	return nil
}

// FindSequenses метод для поиска прав на просмотр
func FindSequenses(ctx context.Context) error {
	foundCasbinRule, err := role_model.GetAllPrivileges()
	if err != nil {
		return err
	}

	sequences := make(map[int64]map[int64]struct{})
	if len(foundCasbinRule) == 0 {
		log.Debug("Not found casbin rule.")
	} else {
		// репозитории и пользователи
		for _, rule := range foundCasbinRule {
			if _, ok := sequences[rule.Org.ID]; !ok {
				sequences[rule.Org.ID] = make(map[int64]struct{})
			}
			sequences[rule.Org.ID][rule.User.ID] = struct{}{}
		}
	}
	return AddCasbinRule(ctx, sequences)
}

// AddCasbinRule метод для добавления ролей casbin rule
func AddCasbinRule(ctx context.Context, sequences map[int64]map[int64]struct{}) error {
	var orgs []*organization.Organization
	// Находим список организаций.
	// todo при деактивации пользователя и при удалении.
	err := db.GetEngine(ctx).
		Where("is_active").
		And("type = ?", user.UserTypeOrganization).
		Find(&orgs)
	if err != nil {
		log.Error("Error found is_active organization: %v", err)
		return err
	}

	var defaultTenant *tenant.ScTenant

	if defaultTenant, err = tenant.GetDefaultTenant(ctx); err != nil {
		log.Error("Error found default tenant: %v", err)
		return err
	}

	for _, org := range orgs {
		if setting.IAM.Enabled {
			_, err = tenant.GetTenantOrganizationsByOrgId(ctx, org.ID)
			if err == nil {
				log.Debug("Found tenant organization with id: %d. Skip adding casbin rule for organization", org.ID)
				continue
			} else if !tenant.IsErrTenantOrganizationNotExists(err) {
				log.Error("Error found tenant organization: %v", err)
				return err
			}
			log.Debug("Not found tenant organization with id: %d. Add casbin rule for organization", org.ID)
		}
		allOrgMembers, _, err := org.GetMembers()
		if err != nil {
			log.Error("Error found all org members: %v", err)
			return err
		}

		ownerTeam, err := organization.GetOwnerTeam(ctx, org.ID)
		if err != nil {
			log.Error("Error found owner team: %v", err)
			return err
		}

		err = ownerTeam.LoadMembers(ctx)
		if err != nil {
			log.Error("Error found owner team members: %v", err)
			return err
		}

		for _, member := range allOrgMembers {
			var targetRule = role_model.READER
			if _, ok := sequences[org.ID][member.ID]; ok {
				continue
			}

			for _, owner := range ownerTeam.Members {
				if owner.ID == member.ID {
					targetRule = role_model.OWNER
					break
				}
			}

			err = role_model.GrantUserPermissionToOrganization(
				member,
				defaultTenant.ID,
				org,
				targetRule,
			)
			if err != nil {
				return err
			}
		}

		if org.Visibility == structs.VisibleTypeLimited {
			if err = role_model.AddProjectToInnerSource(org); err != nil {
				return err
			}
		}
	}
	return nil
}
