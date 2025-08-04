package hooks

import (
	"fmt"
	"net/http"

	repo_model "code.gitea.io/gitea/models/repo"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
)

func LoadRepository(ctx *gitea_context.PrivateContext, ownerName, repoName string) *repo_model.Repository {
	repo, err := repo_model.GetRepositoryByOwnerAndName(ctx, ownerName, repoName)
	if err != nil {
		log.Error("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Failed to get repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return nil
	}
	if repo.OwnerName == "" {
		repo.OwnerName = ownerName
	}
	return repo
}
