package create_default

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/log"
	"context"
	"github.com/google/uuid"
	"strings"
)

// CreateLicensesInfo заполняем таблицу sc_licenses_info из папки licenses
func CreateLicensesInfo(ctx context.Context) error {
	var licensesInfo []repo.ScLicensesInfo
	for _, licenseInformation := range informationAboutLicensesFromGitHub {
		licInfo := repo.ScLicensesInfo{
			ID:          uuid.NewString(),
			SpdxID:      licenseInformation.SpdxId,
			Title:       licenseInformation.Name,
			Description: licenseInformation.Description,
			Permissions: strings.Join(licenseInformation.Permissions, ","),
			Conditions:  strings.Join(licenseInformation.Conditions, ","),
			Limitations: strings.Join(licenseInformation.Limitations, ","),
			Body:        licenseInformation.Body,
		}
		licensesInfo = append(licensesInfo, licInfo)
	}
	has, err := db.GetEngine(ctx).Table("sc_licenses_info").Get(&repo.ScLicensesInfo{})
	if err != nil {
		log.Error("CreateLicensesInfo failed while getting some rows from sc_licenses_info: %v", err)
		return err
	}
	if !has {
		if _, errInsertLicenseInfo := db.GetEngine(ctx).Insert(licensesInfo); errInsertLicenseInfo != nil {
			log.Error("CreateLicensesInfo failed while adding information about licenses: %v", err)
			return errInsertLicenseInfo
		}
	}
	return nil
}
