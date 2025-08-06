package context

import (
	gitModel "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/perm/access"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	goContext "context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func RepoAssigmentByAuthSbt() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		ctx.Repo.Repository = repoAssigmentSbt(ctx)
	}
}

func repoAssigmentSbt(ctx *context.Context) (contextRepo *repoModel.Repository) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	repoName := ctx.Params(":reponame")

	repo, err := repoModel.GetRepositoryByName(ctx.Doer.ID, repoName)
	if err != nil {
		if repoModel.IsErrRepoNotExist(err) {
			log.Debug("Repo with repoName: %s doesn't exist for userName: %s", repoName, ctx.Doer.Name)
			ctx.JSON(http.StatusBadRequest, apiError.RepoDoesNotExist(ctx.Doer.Name, repoName))

			return
		}

		log.Error("Error while getting repository by name: %s for user: %s. Error: %s", repoName, ctx.Doer.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	return repo
}

// GitRepoAssigmentSbt Получение данных о репозитории, гит репозитории и прав на репозиторий (для владельца репозитория)
// по владельцу и имени репозитория
//
//	Если пользователь не авторизован то он может получить данные только о публичном репозитории любого пользователя
//	Если пользователь авторизован то он может получить данные о публичном репозитории любого пользователя или о своем приватном
func GitRepoAssigmentSbt(ctx *context.Context) (cancel goContext.CancelFunc) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	RepoAssigmentSbt(ctx)

	if ctx.Written() {
		return
	}
	// Получить права на различные действия в репозитории, только для владельца репозитория
	if ctx.IsSigned && ctx.Repo.Owner.LowerName == ctx.Doer.LowerName {
		ctx.Repo.Permission.AccessMode = perm.AccessModeOwner
		if err := ctx.Repo.Repository.LoadUnits(ctx); err != nil {
			log.Error("Error has occurred while getting permission for repository: %s. Error: %v", ctx.Repo.Repository.FullName(), err)

			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		ctx.Repo.Permission.Units = ctx.Repo.Repository.Units
		ctx.Repo.Permission.UnitsMode = make(map[unit.Type]perm.AccessMode)
		for _, u := range ctx.Repo.Repository.Units {
			ctx.Repo.Permission.UnitsMode[u.Type] = ctx.Repo.Permission.AccessMode
		}
	}

	gitRepo, err := git.OpenRepository(ctx, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, ctx.Repo.Repository.RepoPath())
	if err != nil {
		if strings.Contains(err.Error(), "repository does not exist") || strings.Contains(err.Error(), "no such file or directory") {
			ctx.Repo.Repository.MarkAsBrokenEmpty()
			log.Debug("Repository %s has a broken git repository on the file system: %s.", ctx.Repo.Repository.FullName(), ctx.Repo.Repository.RepoPath())

			ctx.JSON(http.StatusBadRequest, apiError.GitRepoDoesNotExist(ctx.Repo.Repository.RepoPath()))
			return
		}
		log.Error("Unknown error has occurred while opening git repository by repoPath: %s. Error: %v", ctx.Repo.Repository.RepoPath(), err)

		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if ctx.Repo.GitRepo != nil {
		ctx.Repo.GitRepo.Close()
	}
	ctx.Repo.GitRepo = gitRepo

	// We opened it, we should close it
	cancel = func() {
		// If it's been set to nil then assume someone else has closed it.
		if ctx.Repo.GitRepo != nil {
			ctx.Repo.GitRepo.Close()
		}
	}

	return cancel
}

// RepoAssigmentSbt Получение данных о репозитории по владельцу и имени репозитория
//
//	Если пользователь не авторизован то он может получить данные только о публичном репозитории любого пользователя
//	Если пользователь авторизован то он может получить данные о публичном репозитории любого пользователя или о своем приватном
func RepoAssigmentSbt(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	ownerName := ctx.Params(":username")
	repoName := ctx.Params(":reponame")

	u, err := userModel.GetUserByName(ctx, ownerName)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("Error has occurred while get repo name: %s/%s, owner with name: %s not found", ownerName, repoName, ownerName)

			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(ownerName))
			return
		}
		log.Error("Error has occurred while getting user with name: %s, error: %v", ownerName, err)

		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	repo, err := repoModel.GetRepositoryByName(u.ID, repoName)
	if err != nil {
		if repoModel.IsErrRepoNotExist(err) { //если репозиторий не существует
			log.Debug("Repo with repoName: %s/%s doesn't exist for userName: %s", ownerName, repoName, u.Name)

			ctx.JSON(http.StatusBadRequest, apiError.RepoDoesNotExist(u.Name, repoName))
			return
		}

		log.Error("Error while getting repository by repoName: %s/%s for user: %s. Error: %s", ownerName, repoName, u.Name, err)

		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	// добавим в контекст найденного владельца репы
	ctx.Repo.Owner = u
	repo.Owner = u
	ctx.Repo.Permission, _ = access.GetUserRepoPermission(ctx, repo, ctx.Doer)

	// Если пользователь авторизован и является владельцем (даже если репозиторий принадлежит организации),
	// то вернем данные репозитория в не зависимости публичное оно или приватное
	if ctx.IsSigned {
		if ctx.Repo.Permission.HasAccess() {
			ctx.Repo.Repository = repo

			return
		} else {
			// Если пользователь не владелец и репозиторий приватный, то вернем ошибку
			if repo.IsPrivate {
				log.Debug("Get repo with repoName: %s/%s is not allowed because it is private and user: %s is not owner", ownerName, repoName, ctx.Doer.Name)

				ctx.JSON(http.StatusNotFound, apiError.RepoDoesNotExist(ownerName, repoName))
				return
			}
		}
	} else {
		// Если пользователь не авторизован и репозиторий приватный, то вернем ошибку
		if repo.IsPrivate {
			log.Debug("Get repo with repoName: %s/%s is not allowed because it is private and user is not authorized", ownerName, repoName)

			ctx.JSON(http.StatusNotFound, apiError.RepoDoesNotExist(ownerName, repoName))
			return
		}
	}

	// Если репозиторий публичный то вернем его данные
	ctx.Repo.Repository = repo
}

// RepoRefByTypeSbt handles repository reference name for a specific type
// of repository reference see modules/context/repo.go#RepoRefByType
func RepoRefByTypeSbt(refType context.RepoRefType, ignoreNotExistErr ...bool) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		repoRefByTypeSbt(ctx, refType, ignoreNotExistErr...)
	}
}

// RepoRefByTypeSbt handles repository reference name for a specific type
// of repository reference see modules/context/repo.go#RepoRefByType
func repoRefByTypeSbt(ctx *context.Context, refType context.RepoRefType, ignoreNotExistErr ...bool) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	// Empty repository does not have reference information.
	if ctx.Repo.Repository.IsEmpty {
		// assume the user is viewing the (non-existent) default branch
		ctx.Repo.IsViewBranch = true
		ctx.Repo.BranchName = ctx.Repo.Repository.DefaultBranch
		ctx.Data["TreePath"] = ""
		return
	}

	var (
		refName string
		err     error
	)

	if ctx.Repo.GitRepo == nil {
		repoPath := repoModel.RepoPath(ctx.Repo.Owner.Name, ctx.Repo.Repository.Name)
		ctx.Repo.GitRepo, err = git.OpenRepository(ctx, ctx.Repo.Owner.Name, ctx.Repo.Repository.Name, repoPath)
		if err != nil {
			log.Error("Invalid repo: %s , error: %v", repoPath, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
		// We opened it, we should close it
		defer func() {
			// If it's been set to nil then assume someone else has closed it.
			if ctx.Repo.GitRepo != nil {
				ctx.Repo.GitRepo.Close()
			}
		}()
	}

	// Get default branch.
	if len(ctx.Params("*")) == 0 {
		refName = ctx.Repo.Repository.DefaultBranch
		if !ctx.Repo.GitRepo.IsBranchExist(refName) {
			brs, _, err := ctx.Repo.GitRepo.GetBranchNames(0, 0)
			if err == nil && len(brs) != 0 {
				refName = brs[0]
			} else if len(brs) == 0 {
				log.Error("No branches in non-empty repository %s", ctx.Repo.GitRepo.Path)
				ctx.Repo.Repository.MarkAsBrokenEmpty()
			} else {
				log.Error("GetBranches error: %v", err)
				ctx.Repo.Repository.MarkAsBrokenEmpty()
			}
		}
		ctx.Repo.RefName = refName
		ctx.Repo.BranchName = refName
		ctx.Repo.Commit, err = ctx.Repo.GitRepo.GetBranchCommit(refName)
		if err == nil {
			ctx.Repo.CommitID = ctx.Repo.Commit.ID.String()
		} else if strings.Contains(err.Error(), "fatal: not a git repository") || strings.Contains(err.Error(), "object does not exist") {
			// if the repository is broken, we can continue to the handler code, to show "Settings -> Delete Repository" for end users
			log.Error("Unable to GetBranchCommit error: %v", err)
			ctx.Repo.Repository.MarkAsBrokenEmpty()
		} else {
			log.Error("An error has occurred while to try GetBranchCommit with error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
		ctx.Repo.IsViewBranch = true
	} else {
		refName = getRefName(ctx.Base, ctx.Repo, refType, log)
		ctx.Repo.RefName = refName
		isRenamedBranch, has := ctx.Data["IsRenamedBranch"].(bool)
		if isRenamedBranch && has {
			renamedBranchName := ctx.Data["RenamedBranchName"].(string)
			ctx.Flash.Info(ctx.Tr("repo.branch.renamed", refName, renamedBranchName))
			link := setting.AppSubURL + strings.Replace(ctx.Req.URL.EscapedPath(), util.PathEscapeSegments(refName), util.PathEscapeSegments(renamedBranchName), 1)
			// todo Нужно разобраться что тут происходит и убрать все не нужное
			ctx.Redirect(link)
			return
		}

		if refType.RefTypeIncludesBranches() && ctx.Repo.GitRepo.IsBranchExist(refName) {
			ctx.Repo.IsViewBranch = true
			ctx.Repo.BranchName = refName

			ctx.Repo.Commit, err = ctx.Repo.GitRepo.GetBranchCommit(refName)
			if err != nil {
				log.Error("An error has occurred while to try GetBranchCommit with error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

				return
			}
			ctx.Repo.CommitID = ctx.Repo.Commit.ID.String()

		} else if refType.RefTypeIncludesTags() && ctx.Repo.GitRepo.IsTagExist(refName) {
			ctx.Repo.IsViewTag = true
			ctx.Repo.TagName = refName

			ctx.Repo.Commit, err = ctx.Repo.GitRepo.GetTagCommit(refName)
			if err != nil {
				if git.IsErrNotExist(err) {
					log.Debug("Unable to get commit by Tag: %s, error: %v", refName, err)
					ctx.JSON(http.StatusBadRequest, apiError.CommitNotExist(refName))

					return
				}
				log.Error("An error has occurred while to try GetTagCommit with error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

				return
			}
			ctx.Repo.CommitID = ctx.Repo.Commit.ID.String()
		} else if len(refName) >= 7 && len(refName) <= git.SHAFullLength {
			ctx.Repo.IsViewCommit = true
			ctx.Repo.CommitID = refName

			ctx.Repo.Commit, err = ctx.Repo.GitRepo.GetCommit(refName)
			if err != nil {
				log.Debug("Commit not found by refName: %s in repo: %s", refName, ctx.Repo.Repository.Name)
				ctx.JSON(http.StatusBadRequest, apiError.CommitNotExist(refName))

				return
			}
			// If short commit ID add canonical link header
			if len(refName) < git.SHAFullLength {
				ctx.RespHeader().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"",
					util.URLJoin(setting.AppURL, strings.Replace(ctx.Req.URL.RequestURI(), util.PathEscapeSegments(refName), url.PathEscape(ctx.Repo.Commit.ID.String()), 1))))
			}
		} else {
			if len(ignoreNotExistErr) > 0 && ignoreNotExistErr[0] {
				return
			}
			log.Debug("Branch or tag not exist: %s in repo: %s", refName, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(refName))

			return
		}
	}

	ctx.Repo.CommitsCount, err = ctx.Repo.GetCommitsCount()
	if err != nil {
		log.Error("An error has occurred while try get commit count of repo: %s, error: %v", ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}
	ctx.Repo.GitRepo.LastCommitCache = git.NewLastCommitCache(ctx.Repo.CommitsCount, ctx.Repo.Repository.FullName(), ctx.Repo.GitRepo, cache.GetCache())
}

// getRefName Так как в качестве ссылки может как имя ветки так и тэг, ид коммита или ссылка на blob то в этом
// методе происходит получение ссылки конкретного вида
func getRefName(ctx *context.Base, repo *context.Repository, pathType context.RepoRefType, log logger.Logger) string {
	path := ctx.Params("*")
	switch pathType {
	case context.RepoRefLegacy, context.RepoRefAny:
		if refName := getRefName(ctx, repo, context.RepoRefBranch, log); len(refName) > 0 {
			return refName
		}
		if refName := getRefName(ctx, repo, context.RepoRefTag, log); len(refName) > 0 {
			return refName
		}
		// For legacy and API support only full commit sha
		parts := strings.Split(path, "/")
		if len(parts) > 0 && len(parts[0]) == git.SHAFullLength {
			repo.TreePath = strings.Join(parts[1:], "/")
			return parts[0]
		}
		if refName := getRefName(ctx, repo, context.RepoRefBlob, log); len(refName) > 0 {
			return refName
		}
		repo.TreePath = path
		return repo.Repository.DefaultBranch
	case context.RepoRefBranch:
		ref := getRefNameFromPath(repo, path, repo.GitRepo.IsBranchExist)
		if len(ref) == 0 {
			// maybe it's a renamed branch
			return getRefNameFromPath(repo, path, func(s string) bool {
				b, exist, err := gitModel.FindRenamedBranch(ctx, repo.Repository.ID, s)
				if err != nil {
					log.Debug("Unable to find renamed branch in repo with id: %s, error: %v", repo.Repository.ID, err)

					return false
				}

				if !exist {
					return false
				}
				log.Debug("Found renamed branch: %s in repo with id: %s", b, repo.Repository.ID)

				return true
			})
		}

		return ref
	case context.RepoRefTag:
		return getRefNameFromPath(repo, path, repo.GitRepo.IsTagExist)
	case context.RepoRefCommit:
		parts := strings.Split(path, "/")
		if len(parts) > 0 && len(parts[0]) >= 7 && len(parts[0]) <= git.SHAFullLength {
			repo.TreePath = strings.Join(parts[1:], "/")
			return parts[0]
		}
	case context.RepoRefBlob:
		_, err := repo.GitRepo.GetBlob(path)
		if err != nil {
			return ""
		}
		return path
	default:
		log.Debug("Unrecognized path type: %v", path)
	}
	return ""
}

// getRefNameFromPath вспомогательная функция для разбора пути до файла
func getRefNameFromPath(repo *context.Repository, path string, isExist func(string) bool) string {
	refName := ""
	parts := strings.Split(path, "/")
	for i, part := range parts {
		refName = strings.TrimPrefix(refName+"/"+part, "/")
		if isExist(refName) {
			repo.TreePath = strings.Join(parts[i+1:], "/")
			return refName
		}
	}
	return ""
}
