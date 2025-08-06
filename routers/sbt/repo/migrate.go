package repo

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/lfs"
	repoModule "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/migrations"
	"net/http"
	"strings"
)

func Migrate(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.MigrateRepo)

	var (
		repoOwner *userModel.User
		err       error
	)

	remoteAddr, err := forms.ParseRemoteAddr(req.CloneAddr, req.AuthUsername, req.AuthPassword)
	if err == nil {
		err = migrations.IsMigrateURLAllowed(remoteAddr, ctx.Doer)
	}
	if err != nil {
		handleRemoteAddrError(ctx, remoteAddr, err, log)
		return
	}

	//проверяем настройки
	if req.Mirror && setting.Mirror.DisableNewPull {
		log.Debug("Can not migrate repository %s, the site administrator has disabled the creation of new pull mirrors", remoteAddr)

		ctx.JSON(http.StatusForbidden, apiError.RepoMigrationProhibited("The site administrator has disabled creation of new pull mirrors."))
		return
	}

	if setting.Repository.DisableMigrations {
		log.Debug("Can not migrate repository %s, the site administrator has disabled migrations.", remoteAddr)

		ctx.JSON(http.StatusForbidden, apiError.RepoMigrationProhibited("The site administrator has disabled migrations."))
		return
	}

	// проверяем владельца
	if len(req.RepoOwner) != 0 {
		repoOwner, err = userModel.GetUserByName(ctx, req.RepoOwner)
	} else {
		repoOwner = ctx.Doer
	}
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("An error occurred while migrating repository %s. Owner is not exist", remoteAddr)

			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(req.RepoOwner))
		} else {
			log.Error("An error occurred while migrating repository %s, err: %v", remoteAddr, err)

			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if !ctx.Doer.IsAdmin {
		if !repoOwner.IsOrganization() && ctx.Doer.ID != repoOwner.ID {
			log.Debug("Can not migrate repository %s, provided user: %s is not an organization", remoteAddr, repoOwner.Name)

			ctx.JSON(http.StatusBadRequest, apiError.UserNotOrganization())
			return
		}

		if repoOwner.IsOrganization() {
			// Проверка прав на организацию.
			isOwner, err := organization.OrgFromUser(repoOwner).IsOwnedBy(ctx.Doer.ID)
			if err != nil {
				log.Error("An error occurred while migrating repository %s, err: %v", remoteAddr, err)

				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			} else if !isOwner {
				log.Debug("Can not migrate repository %s, user : %s is not owner of it", remoteAddr, ctx.Doer.Name)

				ctx.JSON(http.StatusForbidden, apiError.UserIsNotOwner())
				return
			}
		}
	}

	gitServiceType := convert.ToGitServiceType(req.Service)

	req.LFS = req.LFS && setting.LFS.StartServer

	if req.LFS && len(req.LFSEndpoint) > 0 {
		ep := lfs.DetermineEndpoint("", req.LFSEndpoint)
		if ep == nil {
			log.Error("An error occurred while migrating repository %s,invalid LFS endpoint", remoteAddr)

			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		err = migrations.IsMigrateURLAllowed(ep.String(), ctx.Doer)
		if err != nil {
			handleRemoteAddrError(ctx, ep.String(), err, log)
			return
		}
	}

	opts := migrations.MigrateOptions{
		CloneAddr:      remoteAddr,
		RepoName:       req.RepoName,
		Description:    req.Description,
		Private:        req.Private || setting.Repository.ForcePrivate,
		Mirror:         req.Mirror,
		LFS:            req.LFS,
		LFSEndpoint:    req.LFSEndpoint,
		AuthUsername:   req.AuthUsername,
		AuthPassword:   req.AuthPassword,
		AuthToken:      req.AuthToken,
		Wiki:           req.Wiki,
		Issues:         req.Issues,
		Milestones:     req.Milestones,
		Labels:         req.Labels,
		Comments:       req.Issues || req.PullRequests,
		PullRequests:   req.PullRequests,
		Releases:       req.Releases,
		GitServiceType: gitServiceType,
		MirrorInterval: req.MirrorInterval,
	}
	if opts.Mirror {
		opts.Issues = false
		opts.Milestones = false
		opts.Labels = false
		opts.Comments = false
		opts.PullRequests = false
		opts.Releases = false
	}

	repo, err := repoModule.CreateRepository(ctx.Doer, repoOwner, repoModule.CreateRepoOptions{
		Name:           opts.RepoName,
		Description:    opts.Description,
		OriginalURL:    req.CloneAddr,
		GitServiceType: gitServiceType,
		IsPrivate:      opts.Private,
		IsMirror:       opts.Mirror,
		Status:         repoModel.RepositoryBeingMigrated,
	})
	if err != nil {
		handleMigrateError(ctx, remoteAddr, err, log)
		return
	}

	opts.MigrateToRepoID = repo.ID

	if repo, err = migrations.MigrateRepository(graceful.GetManager().HammerContext(), ctx.Doer, repoOwner.Name, opts, nil); err != nil {
		handleMigrateError(ctx, remoteAddr, err, log)
		return
	}

	ctx.JSON(http.StatusCreated, request.Repo{ID: repo.ID})

	cache.RemoveItem(sbtCache.GenerateRepoListKey(ctx.Doer.Name) + "*")
}

func handleMigrateError(ctx *context.Context, remoteAddr string, err error, log logger.Logger) {
	switch {
	case repoModel.IsErrRepoAlreadyExist(err):
		log.Debug("Can not migrate repository %s. Repository already exist.", remoteAddr)

		ctx.JSON(http.StatusBadRequest, apiError.RepoAlreadyExists())
	case repoModel.IsErrRepoFilesAlreadyExist(err):
		log.Debug("Can not migrate repository %s. Repository not empty.", remoteAddr)

		ctx.JSON(http.StatusBadRequest, apiError.RepoNotEmpty())

	case repoModel.IsErrCreateUserRepo(err):
		log.Debug("Can not migrate repository %s. Creating a repository outside the project is prohibited", remoteAddr)
		ctx.JSON(http.StatusBadRequest, apiError.UserRepoCreate())

	case db.IsErrNameReserved(err):
		log.Debug("Can not migrate repository %s because name: %s is reserved.", remoteAddr, err.(db.ErrNameReserved).Name)
		ctx.JSON(http.StatusBadRequest, apiError.RepoWrongName())

	case db.IsErrNamePatternNotAllowed(err):
		log.Debug("Can not migrate repository %s because repoName pattern: %s not allowed.", remoteAddr, err.(db.ErrNamePatternNotAllowed).Pattern)
		ctx.JSON(http.StatusBadRequest, apiError.RepoWrongName())

	default:
		err = util.SanitizeErrorCredentialURLs(err)
		if strings.Contains(err.Error(), "Authentication failed") ||
			strings.Contains(err.Error(), "Bad credentials") ||
			strings.Contains(err.Error(), "could not read Username") {
			log.Debug("Can not migrate repository %s. Bad credentials.", remoteAddr)

			ctx.JSON(http.StatusBadRequest, apiError.RemoteRepoAuthFailed(err.Error()))
		} else {
			log.Error("Can not migrate repository %s. Err: %v", remoteAddr, err)

			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
	}
}

func handleRemoteAddrError(ctx *context.Context, address string, err error, log logger.Logger) {
	if models.IsErrInvalidCloneAddr(err) {
		addrErr := err.(*models.ErrInvalidCloneAddr)
		log.Debug("Unprocessable address: %s error: %v while migrating repository", address, addrErr)

		ctx.JSON(http.StatusBadRequest, apiError.UnprocessableRemoteRepoAddress())

	} else {
		log.Error("Can not migrate repository %s, provided repository address is wrong. Err: %v", address, err)

		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}
}
