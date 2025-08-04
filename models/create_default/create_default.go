package create_default

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/user"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/timeutil"
	"context"
	"github.com/google/uuid"
	"xorm.io/builder"
)

// CreateDefault создание тенантов и присвоение рганизаций по умолчанию.
func CreateDefault(ctx context.Context) error {
	scTenant, err := CreateTenant(ctx)
	if err != nil {
		return err
	}
	err = StandardizationProjectVisibility(ctx)
	if err != nil {
		return err
	}

	organizations, err := FindOrganisation(ctx)
	if err != nil {
		return err
	}
	if organizations == nil {
		return nil
	}

	var scTenantOrganizations []*tenant.ScTenantOrganizations

	// Находим список организаций.
	if err = db.GetEngine(ctx).
		Find(&scTenantOrganizations); err != nil {
		return err
	}

	foundRecordSCTenantOrganization := make(map[int64]struct{})
	// Создаем map для организаций, для сравнений в дальнейшем.
	for _, scTenantOrganization := range scTenantOrganizations {
		foundRecordSCTenantOrganization[scTenantOrganization.OrganizationID] = struct{}{}
	}

	tenantOrganizations := make([]*tenant.ScTenantOrganizations, 0, len(organizations))
	for _, project := range organizations {
		if _, ok := foundRecordSCTenantOrganization[project.ID]; ok {
			continue
		}
		tenantOrganizations = append(tenantOrganizations, &tenant.ScTenantOrganizations{
			ID:             uuid.NewString(),
			TenantID:       scTenant.ID,
			OrganizationID: project.ID,
		},
		)
	}
	if len(tenantOrganizations) == 0 {
		return nil
	}
	if _, err := db.GetEngine(ctx).Insert(&tenantOrganizations); err != nil {
		log.Error("insert tenant organizations: %v", err)
		return err
	}
	return nil
}

// CreateTenant проверяет на наличие тенанта, при отсутствии создает.
func CreateTenant(ctx context.Context) (scTenant *tenant.ScTenant, err error) {
	foundTenant := tenant.ScTenant{Default: true}
	// Находим список тенантов.
	checkTenant, err := db.GetEngine(ctx).Get(&foundTenant)
	if err != nil {
		log.Error("Not enough tenant is default: %v", err)
		return scTenant, err
	}
	defaultTenantName := "tenant"
	// Если тенантов нет, то создать.
	if !checkTenant {
		scTenant = &tenant.ScTenant{
			ID:        uuid.NewString(),
			Name:      defaultTenantName,
			Default:   true,
			IsActive:  true,
			CreatedAt: timeutil.TimeStampNow(),
			UpdatedAt: timeutil.TimeStampNow(),
			OrgKey:    defaultTenantName,
		}
		scTenant.ID = uuid.NewString()
		_, err = db.GetEngine(ctx).Insert(scTenant)
		if err != nil {
			log.Error("Error insert tenant: %v", err)
			return scTenant, err
		}
		return scTenant, nil
	} else {
		return &foundTenant, nil
	}
}

// StandardizationProjectVisibility изменяем видимость проектов..
func StandardizationProjectVisibility(ctx context.Context) error {
	_, err := db.GetEngine(ctx).
		Where(builder.Eq{"type": 1, "visibility": 0}).
		Cols("visibility").
		Update(&organization.Organization{Visibility: structs.VisibleTypeLimited})
	if err != nil {
		log.Error("update visibility: %v", err)
		return err
	}
	return nil
}

// FindOrganisation возвращает пользователей.
func FindOrganisation(ctx context.Context) (organizations []*user_model.User, err error) {
	// Находим список пользователей.
	err = db.GetEngine(ctx).
		Where("type = ?", user.UserTypeOrganization).
		Find(&organizations)

	if err != nil {
		log.Error("found an organizations is failed: %v", err)
		return organizations, err
	}
	// Если организаций нет, то выходим.
	if len(organizations) == 0 {
		log.Debug("organizations not found")
		return nil, nil
	}
	return organizations, nil
}
