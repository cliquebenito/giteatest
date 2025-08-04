package repo

import (
	"code.gitea.io/gitea/models/license"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"fmt"
	"net/http"
	"strings"
)

// GetLicenseInfo endpoint для отображения информации о лицензии
func GetLicenseInfo(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.GetLicenseInfoForm)
	repoLicenses, err := repo.GetLicenseInfoByCommitIDAndBranch(ctx, form.RepositoryID, form.PathFile, form.Branch)
	if err != nil {
		log.Error("LicenseInfo repo.GetLicenseInfoByCommitIDAndBranch failed while get licences info: %v", err)
		ctx.Error(http.StatusNotFound, fmt.Sprintf("LicenseInfo repo.GetLicenseInfoByCommitIDAndBranch failed because license is not in repository_id %v: %v", form.RepositoryID, err))
		return
	}
	spdxIDLicenses := make([]string, len(repoLicenses))
	for idx, repLicense := range repoLicenses {
		spdxIDLicenses[idx] = repLicense.SpdxID
	}
	licenseInfos, err := repo.GetInfoAboutLicenses(ctx, spdxIDLicenses)
	if err != nil {
		log.Error("LicenseInfo repo.GetInfoAboutLicenses failed while getting info about licences: %v", err)
		ctx.Error(http.StatusInternalServerError, fmt.Sprintf("LicenseInfo repo.GetInfoAboutLicenses failed theres is not such license in db: %v", err))
		return
	}
	if len(licenseInfos) == 0 {
		ctx.JSON(http.StatusNotFound, &license.ResponseInfoLicense{})
		return
	}
	licenseInfo := licenseInfos[0]
	ctx.JSON(http.StatusOK, &license.ResponseInfoLicense{
		Name:        licenseInfo.Title,
		Description: licenseInfo.Description,
		Permissions: strings.Split(licenseInfo.Permissions, ","),
		Conditions:  strings.Split(licenseInfo.Conditions, ","),
		Limitations: strings.Split(licenseInfo.Limitations, ","),
	})
}
