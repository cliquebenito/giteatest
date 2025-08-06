package git

import (
	"regexp"
	"strings"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

var (
	// Компилируем регулярное выражение для распознавания файла с потенциальным возможным содержанием лицензии
	regex = regexp.MustCompile(`(?i)(legal|copy(left|right|ing)|apache|mit|(un)?li[cs]en[cs]e(s?))(\.(md|txt|html|rst))?$`)
)

// CheckLicenseInfo  получение лицензии из репозитория
func (repo *Repository) CheckLicenseInfo(commitID string) (map[string][]string, error) {
	commit, err := repo.GetCommit(commitID)
	if err != nil {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, err
	}
	tree := commit.Tree
	entries, err := tree.ListEntriesRecursiveWithSize()
	if err != nil {
		log.Error("CheckLicenseInfo tree.ListEntriesRecursiveWithSize failed while returning subtree for tree_id %s: %v", tree.ID.String(), err)
		return nil, err
	}

	fileNameLicense := make(map[string][]string)
	for _, f := range entries {
		select {
		case <-repo.Ctx.Done():
			return nil, repo.Ctx.Err()
		default:
		}
		if f.IsDir() || strings.Contains(f.Name(), "/") {
			continue
		}
		if !regex.MatchString(strings.ToLower(f.Name())) {
			continue
		}

		blobClient, err := repo.BlobClient.GetBlob(repo.Ctx, &gitalypb.GetBlobRequest{Repository: repo.GitalyRepo, Oid: f.ID.String(), Limit: -1})
		if err != nil {
			return nil, err
		}

		resp := make([]byte, 0, fileSizeLimit)
		canRead := true
		for canRead {
			blobResponse, _ := blobClient.Recv()
			if blobResponse == nil {
				canRead = false
			} else {
				resp = append(resp, blobResponse.Data...)
			}
		}

		res := licensedb.InvestigateLicenseText(resp)
		if res != nil {
			for licenseName := range res {
				fileNameLicense[f.Name()] = append(fileNameLicense[f.Name()], licenseName)
			}
		}
	}
	return fileNameLicense, nil
}
