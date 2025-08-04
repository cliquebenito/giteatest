package repo

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

/*
GetRepoCommitsList метод получения списка коммитов репозитория
todo: добавить проверку на валидность и наличие sha, path, not
sha - хэш коммита или ветка репозитория
path - путь до файла
not - за исключением коммита или ветки

Bool- переменные по дефолту они false
stat - статистика коммита
files - список файлов коммита
*/
func GetRepoCommitsList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	pageSize := convert.ToCorrectPageSize(ctx.FormInt("limit"))

	listOptions := db.ListOptions{
		PageSize: pageSize,
		Page:     page,
	}

	if listOptions.PageSize > setting.Git.CommitsRangeSize {
		listOptions.PageSize = setting.Git.CommitsRangeSize
	}

	sha := ctx.FormString("sha")
	path := ctx.FormString("path")
	not := ctx.FormString("not")

	if !git.IsValidRefPattern(sha) {
		log.Debug("Wrong git reference name sha: %s", sha)
		ctx.JSON(http.StatusBadRequest, apiError.ValidationError{FieldName: sha, ErrorMessage: "Wrong git reference name"})

		return
	}

	var (
		commitsCountTotal int64
		commits           []*git.Commit
		err               error
	)

	// если в запросе не был указан путь до файла
	if len(path) == 0 {
		var baseCommit *git.Commit

		// если в запросе не был указан sha, тогда возвращаем коммит с дефолтовой ветки
		if len(sha) == 0 {
			// no sha supplied - use default branch
			head, err := ctx.Repo.GitRepo.GetHEADBranch()
			if err != nil {
				log.Error("Error has occurred while getting HEAD branch in repoId: %d. Error: %v", ctx.Repo.Repository.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

				return
			}

			baseCommit, err = ctx.Repo.GitRepo.GetBranchCommit(head.Name)
			if err != nil {
				log.Error("Error has occurred while getting branch commit: %s in repoId: %d. Error: %v", head.Name, ctx.Repo.Repository.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

				return
			}
		} else {
			// если в запросе был указан sha, тогда возвращаем коммит с указанного Хеша (ветки)
			baseCommit, err = ctx.Repo.GitRepo.GetCommit(sha)
			if err != nil {
				if git.IsErrNotExist(err) {
					log.Debug("No such SHA: %s in repoId: %d. Error: %v", sha, ctx.Repo.Repository.ID, err)
					ctx.JSON(http.StatusBadRequest, apiError.GitReferenceNotExist(sha))

				} else {
					log.Error("Error has occurred while getting branch commit by SHA: %s in repoId: %d. Error: %v", sha, ctx.Repo.Repository.ID, err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				}
				return
			}
		}

		// Общее число коммитов
		commitsCountTotal, err = git.CommitsCount(ctx.Repo.GitRepo.Ctx, git.CommitsCountOptions{
			RepoPath:     ctx.Repo.GitRepo.Path,
			Not:          not,
			Revision:     []string{baseCommit.ID.String()},
			GitalyRepo:   ctx.Repo.GitRepo.GitalyRepo,
			Ctx:          ctx.Repo.GitRepo.Ctx,
			CommitClient: ctx.Repo.GitRepo.CommitClient,
		})

		if err != nil {
			log.Error("Error has occurred while getting total count of commits in repoId: %d. Error: %v", ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}

		// Коммиты с пагинацией и без пути
		commits, err = baseCommit.CommitsByRange(listOptions.Page, listOptions.PageSize, not)
		if err != nil {
			log.Error("Error has occurred while getting commits in repoId: %d, with listOptions: %v. Error: %v", ctx.Repo.Repository.ID, listOptions, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
	} else { // если в запросе был указан путь до файла

		if len(sha) == 0 {
			sha = ctx.Repo.Repository.DefaultBranch
		}

		// Общее число коммитов
		commitsCountTotal, err = git.CommitsCount(ctx,
			git.CommitsCountOptions{
				RepoPath:     ctx.Repo.GitRepo.Path,
				Not:          not,
				Revision:     []string{sha},
				RelPath:      []string{path},
				GitalyRepo:   ctx.Repo.GitRepo.GitalyRepo,
				Ctx:          ctx.Repo.GitRepo.Ctx,
				CommitClient: ctx.Repo.GitRepo.CommitClient,
			})

		if err != nil {
			log.Error("Error has occurred while getting total count of commits by path: %s in repoId: %d. Error: %v", path, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		} else if commitsCountTotal == 0 {
			log.Debug("There is no commit with path: %s in repoId: %d", path, ctx.Repo.Repository.ID)
			ctx.JSON(http.StatusOK, response.CommitListResult{Total: commitsCountTotal, Data: make([]*response.Commit, 0)})
			return
		}

		// Коммиты с пагинацией и файлом
		commits, err = ctx.Repo.GitRepo.CommitsByFileAndRange(
			git.CommitsByFileAndRangeOptions{
				Revision: sha,
				File:     path,
				Not:      not,
				Page:     listOptions.Page,
			})

		if err != nil {
			log.Error("Error has occurred while getting commits by path: %s in repoId: %d, with listOptions: %v. Error: %v", path, ctx.Repo.Repository.ID, listOptions, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
	}

	commitOptions := sbtConvert.ToCommitOptions{
		Stat:  ctx.FormString("stat") != "" && ctx.FormBool("stat"),
		Files: ctx.FormString("files") != "" && ctx.FormBool("files"),
	}

	responseCommits := make([]*response.Commit, len(commits))

	for i, commit := range commits {
		responseCommits[i], err = sbtConvert.ToResponseCommit(ctx.Repo.GitRepo, commit, commitOptions)
		if err != nil {
			log.Error("Error has occurred while converting git.Commit to response.Commit in repoId: %d. Error: %v", ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
	}

	ctx.JSON(http.StatusOK, response.CommitListResult{Total: commitsCountTotal, Data: responseCommits})
}
