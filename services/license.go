package services

import (
	"fmt"
	"strings"

	license_model "code.gitea.io/gitea/models/license"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
)

// GetLicensesInfoForRepo получает информацию о лицензиях, которые содержатся в репозитории repoID
// на ветке branch после коммита commitID.
func GetLicensesInfoForRepo(repo *git.Repository, commitID string, repoID int64, repoOwnerName, branch string) (map[string]map[license_model.RepoLicenses]struct{}, error) {
	// Получаем информацию о лицензиях, если они есть в репозитории
	fileNameWithLicenses, err := repo.CheckLicenseInfo(commitID)
	if err != nil {
		log.Error("Error has occurred while checking license info for commit '%s': %v", commitID, err)
		return nil, fmt.Errorf("check license info for commit '%s': %w", commitID, err)
	}

	fileLicensesInfo := make(map[string]map[license_model.RepoLicenses]struct{})

	// Если нет лицензий в новом коммите или файл, где хранится лицензия был изменени,
	// удаляем не актуальную информацю о лицензиях из репозитория
	infoAboutLicensesForRepository, err := repo_model.GetLicenseInfoByCommitIDAndBranch(repo.Ctx, repoID, "", branch)
	if err != nil {
		log.Error("Error has occurred while getting license info for commit '%s' and branch '%s': %v", commitID, branch, err)
		return nil, fmt.Errorf("get license info for commit '%s' and branch '%s': %w", commitID, branch, err)
	}
	for _, inf := range infoAboutLicensesForRepository {
		existsFileName := false
		for fileName := range fileLicensesInfo {
			if strings.Contains(inf.PathFile, fileName) {
				existsFileName = true
				break
			}
		}
		if !existsFileName {
			errDeleteExistLicense := repo_model.DeleteExistLicenses(repo.Ctx, repoID, branch, inf.PathFile, inf.SpdxID)
			if errDeleteExistLicense != nil {
				log.Error("Error has occurred while deleting exist file with licenses for repo with ID '%d': %v", repoID, errDeleteExistLicense)
				return nil, fmt.Errorf("delete gile with licenses for repo with ID '%d': %w", repoID, errDeleteExistLicense)
			}
		}
	}

	for fileName, licenseSpdxIDs := range fileNameWithLicenses {
		path := fmt.Sprintf("%s/%s/%s", repoOwnerName, repo.Name, fileName)
		infoAboutLicensesForRepository, err := repo_model.GetLicenseInfoByCommitIDAndBranch(repo.Ctx, repoID, path, branch)
		if err != nil {
			log.Error("Error has occurred while getting license info for commit '%s' and branch '%s' for path '%s': %v", commitID, branch, path, err)
			return nil, fmt.Errorf("get license info for commit '%s' and branch '%s' for path '%s': %w", commitID, branch, path, err)
		}

		existsLicenses := make(map[string]struct{})
		for _, license := range infoAboutLicensesForRepository {
			existsLicenses[license.SpdxID] = struct{}{}
		}

		for idx, spdxID := range licenseSpdxIDs {
			if _, ok := existsLicenses[spdxID]; ok {
				licenseSpdxIDs = append(licenseSpdxIDs[:idx], licenseSpdxIDs[idx+1:]...)
				delete(existsLicenses, spdxID)
			}
		}
		if len(existsLicenses) > 0 {
			for licenseName := range existsLicenses {
				errDeleteExistLicense := repo_model.DeleteExistLicenses(repo.Ctx, repoID, branch, path, licenseName)
				if errDeleteExistLicense != nil {
					log.Error("Error has occurred while deleting license '%s' with path '%s' from repo with ID '%d' from branch '%s': %v", licenseName, path, repoID, branch, errDeleteExistLicense)
					return nil, fmt.Errorf("delete license '%s' with path '%s' from repo with ID '%d' from branch '%s': %w", licenseName, path, repoID, branch, errDeleteExistLicense)
				}
			}
		}

		licensesInfo, errGetInfo := repo_model.GetInfoAboutLicenses(repo.Ctx, licenseSpdxIDs)
		if errGetInfo != nil {
			log.Error("Error has occurred while getting information for licenses: %v", errGetInfo)
			return nil, fmt.Errorf("get information for licenses: %w", errGetInfo)
		}

		licensesInfos := make(map[license_model.RepoLicenses]struct{})
		for _, licInfo := range licensesInfo {
			title := licInfo.Title
			if title == "" {
				title = licInfo.SpdxID
			}
			licenseInfo := license_model.RepoLicenses{
				RepositoryID: repoID,
				SpdxID:       licInfo.SpdxID,
				NameLicense:  title,
				BranchName:   branch,
				PathFile:     path,
			}
			licensesInfos[licenseInfo] = struct{}{}
		}
		fileLicensesInfo[path] = licensesInfos
	}

	return fileLicensesInfo, nil
}
