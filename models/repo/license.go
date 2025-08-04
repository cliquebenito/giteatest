package repo

import (
	"code.gitea.io/gitea/models/db"
	license_model "code.gitea.io/gitea/models/license"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
	"context"
	"github.com/google/uuid"
	"xorm.io/builder"
)

// ScRepoLicenses создаём таблицу sc_repo_licenses
type ScRepoLicenses struct {
	ID           string             `xorm:"pk uuid"`
	SpdxID       string             `xorm:"NOT NULL"`
	RepositoryID int64              `xorm:"NOT NULL"`
	NameLicense  string             `xorm:"not null"`
	BranchName   string             `xorm:"not null"`
	PathFile     string             `xorm:"not null"`
	CommitID     string             `xorm:"not null"`
	UpdatedAt    timeutil.TimeStamp `xorm:"updated"`
}

// ScLicensesInfo создаём таблицу sc_licenses_info
type ScLicensesInfo struct {
	ID          string `xorm:"pk uuid"`
	SpdxID      string `xorm:"JSON TEXT NOT NULL"`
	Title       string `xorm:"JSON TEXT NOT NULL"`
	Description string `xorm:"JSON TEXT NOT NULL"`
	Permissions string `xorm:"JSON TEXT NOT NULL"`
	Conditions  string `xorm:"JSON TEXT NOT NULL"`
	Limitations string `xorm:"JSON TEXT NOT NULL"`
	Body        string `xorm:"JSON TEXT NOT NULL"`
}

// иницилизируем таблицы
func init() {
	db.RegisterModel(new(ScLicensesInfo))
	db.RegisterModel(new(ScRepoLicenses))
}

// UpsertInfoLicense вставляем или удаляем информацию о лицензиях в репозитории
func UpsertInfoLicense(repoID int64, commitID, branchName string, fileNameLicensesInfo map[string]map[license_model.RepoLicenses]struct{}) error {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		log.Error("Error has occurred while getting db.TxContext: %v", err)
		return err
	}
	defer committer.Close()
	for filePath, licenseInfo := range fileNameLicensesInfo {
		oldInfoLicense, errGetLicenseInfo := GetLicenseInfoByCommitIDAndBranch(ctx, repoID, filePath, branchName)
		if errGetLicenseInfo != nil {
			log.Error("UpsertInfoLicense GetLicenseInfoByCommitIDAndBranch failed while getting licences for repository_id %v: %v", repoID, errGetLicenseInfo)
			return errGetLicenseInfo
		}
		upsertInfo := make([]*ScRepoLicenses, 0)
		for _, old := range oldInfoLicense {
			if _, ok := licenseInfo[old]; ok {
				delete(licenseInfo, old)
			}
		}
		for licInfo := range licenseInfo {
			repoLicenseInfo := &ScRepoLicenses{
				ID:           uuid.NewString(),
				SpdxID:       licInfo.SpdxID,
				CommitID:     commitID,
				PathFile:     licInfo.PathFile,
				NameLicense:  licInfo.NameLicense,
				BranchName:   licInfo.BranchName,
				RepositoryID: licInfo.RepositoryID,
				UpdatedAt:    timeutil.TimeStampNow(),
			}
			upsertInfo = append(upsertInfo, repoLicenseInfo)
		}
		if len(upsertInfo) == 0 {
			return nil
		}
		_, err := db.GetEngine(ctx).Insert(upsertInfo)
		if err != nil {
			log.Error("UpsertInfoLicense Insert failed while adding new notes: %v", err)
			return err
		}
	}
	return committer.Commit()
}

// GetLicenseInfoByCommitIDAndBranch  получаем информацию о файлах с лицензиями по repoID, path, branch
func GetLicenseInfoByCommitIDAndBranch(ctx context.Context, repoID int64, path, branch string) ([]license_model.RepoLicenses, error) {
	var licenseInfo []license_model.RepoLicenses
	sess := db.GetEngine(ctx).
		Table("sc_repo_licenses").
		Where("repository_id = ? and branch_name = ?", repoID, branch)
	if path != "" {
		sess.Where("path_file = ?", path)
	}
	err := sess.Find(&licenseInfo)
	if err != nil {
		log.Error("GetLicenseInfoByCommitIDAndBranch failed while finding licences for repository_id %v: %v", repoID, err)
		return nil, err
	}
	return licenseInfo, err
}

// GetInfoAboutLicenses сопоставляем найденные лицензии с информацией о них из таблицы sc_licenses_info
func GetInfoAboutLicenses(ctx context.Context, spdxID []string) ([]license_model.InfoLicenses, error) {
	var licensesInfo []license_model.InfoLicenses
	err := db.GetEngine(ctx).
		Table("sc_licenses_info").
		Where(builder.Eq{"spdx_id": spdxID}).
		Find(&licensesInfo)
	if err != nil {
		log.Error("GetInfoAboutLicenses failed while getting licences for spdx_id %v: %v", spdxID, err)
		return nil, err
	}
	return licensesInfo, err
}

// DeleteExistLicenses удаляем лицензию, в случае изменения ее в последнем коммите
func DeleteExistLicenses(ctx context.Context, repoID int64, branch, path, spdxID string) error {
	_, err := db.GetEngine(ctx).Delete(&ScRepoLicenses{RepositoryID: repoID, BranchName: branch, PathFile: path, SpdxID: spdxID})
	if err != nil {
		log.Error("DeleteExistLicenses failed while deleting repository_id %v: %v", repoID, err)
		return err
	}
	return nil
}
