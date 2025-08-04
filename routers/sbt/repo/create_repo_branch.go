package repo

import (
	"net/http"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	repo_service "code.gitea.io/gitea/services/repository"
)

/*
CreateBranch метод создания ветки в репозитории
*/
func CreateBranch(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.CreateRepoBranch)

	var oldCommit *git.Commit
	var err error

	//	Проверяем что поле old_ref_name пришло не пустым. Если оно пришло не пустым - берем последний коммит с ветки/тега/коммита
	if len(req.OldRefName) > 0 {
		oldCommit, err = ctx.Repo.GitRepo.GetCommit(req.OldRefName)
		if err != nil {
			if git.IsErrNotExist(err) {
				log.Debug("No such reference: %s in repo with repoId: %d. Error message: %v", req.OldRefName, ctx.Repo.Repository.ID, err)
				ctx.JSON(http.StatusBadRequest, apiError.GitReferenceNotExist(req.OldRefName))

			} else {
				log.Error("Error has occurred while getting repo commit by username: %s and repoId: %d. Error message: %v", ctx.Doer.Name, ctx.Repo.Repository.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			}
			return
		}
	} else {
		// Если не было old_ref_name в запросе, тогда берем коммит из ветки по дефолту (скорее всего main)
		oldCommit, err = ctx.Repo.GitRepo.GetBranchCommit(ctx.Repo.Repository.DefaultBranch)
		if err != nil {
			log.Error("Error has occurred while getting branch commit branchName: %s repoId: %d. Error message: %v", ctx.Repo.Repository.DefaultBranch, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
	}

	// Создаем новую ветку на основании полученного коммита
	if err = repo_service.CreateNewBranchFromCommit(ctx, ctx.Doer, ctx.Repo.Repository, oldCommit.ID.String(), req.BranchName); err != nil {
		if models.IsErrTagAlreadyExists(err) {
			log.Debug("The branch with the same tag: %s already exists. Error: %v", err.(models.ErrTagAlreadyExists).TagName, err)
			ctx.JSON(http.StatusBadRequest, apiError.TagAlreadyExist(err.(models.ErrTagAlreadyExists).TagName))

		} else if models.IsErrBranchAlreadyExists(err) || git.IsErrPushOutOfDate(err) {
			log.Debug("Branch: %s already exist in repository: %s. Error: %v", req.BranchName, ctx.Repo.Repository.Name, err)
			ctx.JSON(http.StatusBadRequest, apiError.BranchAlreadyExist(req.BranchName))

		} else if models.IsErrBranchNameConflict(err) {
			log.Debug("The branch with the same name: %s already exists in repository: %s. Error: %v", req.BranchName, ctx.Repo.Repository.Name, err)
			ctx.JSON(http.StatusBadRequest, "The branch with the same name already exists.")

		} else {
			log.Error("Error has occurred while creating new branch with branchName: %s repoId: %d. Error: %v", req.BranchName, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}

	ctx.Status(http.StatusCreated)
}
