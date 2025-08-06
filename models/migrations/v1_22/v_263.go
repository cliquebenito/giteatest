package v1_22

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/modules/structs"
	"xorm.io/builder"
	"xorm.io/xorm"
)

// ChangeOrganizationVisibility изменение видимости проектов в зависимости от параметров SourceControl.Enabled и SourceControl.TenantWithRoleModeEnabled
func ChangeOrganizationVisibility(x *xorm.Engine) error {
	_, err := db.GetEngine(db.DefaultContext).
		Table("user").
		Where(builder.Eq{"type": 1, "visibility": 0}).
		Cols("visibility").
		Update(&organization.Organization{Visibility: structs.VisibleTypeLimited})
	if err != nil {
		return err
	}
	return nil
}
