package convert

import (
	"code.gitea.io/gitea/modules/context"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/perm"
	repoModel "code.gitea.io/gitea/models/repo"
	unitModel "code.gitea.io/gitea/models/unit"
)

// ToRepo конвертирует Repository в response.Repository
func ToRepo(ctx *context.Context, repo *repoModel.Repository, mode perm.AccessMode, log logger.Logger) *response.Repository {
	return innerToRepo(ctx, repo, mode, false, log)
}

func innerToRepo(ctx *context.Context, repo *repoModel.Repository, mode perm.AccessMode, isParent bool, log logger.Logger) *response.Repository {
	var parent *response.Repository

	cloneLink := repo.CloneLink()
	permission := &response.Permission{
		Admin: mode >= perm.AccessModeAdmin,
		Push:  mode >= perm.AccessModeWrite,
		Pull:  mode >= perm.AccessModeRead,
	}
	if !isParent {
		err := repo.GetBaseRepo(ctx)
		if err != nil {
			return nil
		}
		if repo.BaseRepo != nil {
			parent = innerToRepo(ctx, repo.BaseRepo, mode, true, log)
		}
	}

	hasIssues := false
	var externalTracker *response.ExternalTracker
	var internalTracker *response.InternalTracker
	if unit, err := repo.GetUnit(ctx, unitModel.TypeIssues); err == nil {
		config := unit.IssuesConfig()
		hasIssues = true
		internalTracker = &response.InternalTracker{
			EnableTimeTracker:                config.EnableTimetracker,
			AllowOnlyContributorsToTrackTime: config.AllowOnlyContributorsToTrackTime,
			EnableIssueDependencies:          config.EnableDependencies,
		}
	} else if unit, err := repo.GetUnit(ctx, unitModel.TypeExternalTracker); err == nil {
		config := unit.ExternalTrackerConfig()
		hasIssues = true
		externalTracker = &response.ExternalTracker{
			ExternalTrackerURL:           config.ExternalTrackerURL,
			ExternalTrackerFormat:        config.ExternalTrackerFormat,
			ExternalTrackerStyle:         config.ExternalTrackerStyle,
			ExternalTrackerRegexpPattern: config.ExternalTrackerRegexpPattern,
		}
	}
	hasWiki := false
	var externalWiki *response.ExternalWiki
	if _, err := repo.GetUnit(ctx, unitModel.TypeWiki); err == nil {
		hasWiki = true
	} else if unit, err := repo.GetUnit(ctx, unitModel.TypeExternalWiki); err == nil {
		hasWiki = true
		config := unit.ExternalWikiConfig()
		externalWiki = &response.ExternalWiki{
			ExternalWikiURL: config.ExternalWikiURL,
		}
	}
	hasPullRequests := false
	ignoreWhitespaceConflicts := false
	allowMerge := false
	allowRebase := false
	allowRebaseMerge := false
	allowSquash := false
	allowRebaseUpdate := false
	defaultDeleteBranchAfterMerge := false
	defaultMergeStyle := repoModel.MergeStyleMerge
	defaultAllowMaintainerEdit := false
	if unit, err := repo.GetUnit(ctx, unitModel.TypePullRequests); err == nil {
		config := unit.PullRequestsConfig()
		hasPullRequests = true
		ignoreWhitespaceConflicts = config.IgnoreWhitespaceConflicts
		allowMerge = config.AllowMerge
		allowRebase = config.AllowRebase
		allowRebaseMerge = config.AllowRebaseMerge
		allowSquash = config.AllowSquash
		allowRebaseUpdate = config.AllowRebaseUpdate
		defaultDeleteBranchAfterMerge = config.DefaultDeleteBranchAfterMerge
		defaultMergeStyle = config.GetDefaultMergeStyle()
		defaultAllowMaintainerEdit = config.DefaultAllowMaintainerEdit
	}
	hasProjects := false
	if _, err := repo.GetUnit(ctx, unitModel.TypeProjects); err == nil {
		hasProjects = true
	}

	hasReleases := false
	if _, err := repo.GetUnit(ctx, unitModel.TypeReleases); err == nil {
		hasReleases = true
	}

	hasPackages := false
	if _, err := repo.GetUnit(ctx, unitModel.TypePackages); err == nil {
		hasPackages = true
	}

	hasActions := false
	if _, err := repo.GetUnit(ctx, unitModel.TypeActions); err == nil {
		hasActions = true
	}

	if err := repo.LoadOwner(ctx); err != nil {
		return nil
	}

	numReleases, _ := repoModel.GetReleaseCountByRepoID(ctx, repo.ID, repoModel.FindReleasesOptions{IncludeDrafts: false, IncludeTags: false})

	mirrorInterval := ""
	var mirrorUpdated time.Time
	if repo.IsMirror {
		pullMirror, err := repoModel.GetMirrorByRepoID(ctx, repo.ID)
		if err == nil {
			mirrorInterval = pullMirror.Interval.String()
			mirrorUpdated = pullMirror.UpdatedUnix.AsTime()
		}
	}

	var transfer *response.RepoTransfer
	if repo.Status == repoModel.RepositoryPendingTransfer {
		t, err := models.GetPendingRepositoryTransfer(ctx, repo)
		if err != nil && !models.IsErrNoPendingTransfer(err) {
			log.Warn("GetPendingRepositoryTransfer: %v", err)
		} else {
			if err := t.LoadAttributes(ctx); err != nil {
				log.Warn("LoadAttributes of RepoTransfer: %v", err)
			} else {
				transfer = ToRepoTransfer(ctx, t)
			}
		}
	}

	var language string
	if repo.PrimaryLanguage != nil {
		language = repo.PrimaryLanguage.Language
	}

	var isWatching bool
	var isStarring bool
	if ctx.Doer != nil {
		isWatching = repoModel.IsWatching(ctx.Doer.ID, repo.ID)
		isStarring = repoModel.IsStaring(ctx, ctx.Doer.ID, repo.ID)
	}

	return &response.Repository{
		ID:                            repo.ID,
		Owner:                         ToUserWithAccessMode(ctx, repo.Owner, mode),
		Name:                          repo.Name,
		FullName:                      repo.FullName(),
		Description:                   repo.Description,
		Private:                       repo.IsPrivate,
		Template:                      repo.IsTemplate,
		Empty:                         repo.IsEmpty,
		Archived:                      repo.IsArchived,
		Size:                          int(repo.Size / 1024),
		Fork:                          repo.IsFork,
		Parent:                        parent,
		Mirror:                        repo.IsMirror,
		SSHURL:                        cloneLink.SSH,
		CloneURL:                      cloneLink.HTTPS,
		Website:                       repo.Website,
		Language:                      language,
		Stars:                         repo.NumStars,
		Forks:                         repo.NumForks,
		Watchers:                      repo.NumWatches,
		OpenIssues:                    repo.NumOpenIssues,
		OpenPulls:                     repo.NumOpenPulls,
		Releases:                      int(numReleases),
		DefaultBranch:                 repo.DefaultBranch,
		Created:                       repo.CreatedUnix.AsTime(),
		Updated:                       repo.UpdatedUnix.AsTime(),
		ArchivedAt:                    repo.ArchivedUnix.AsTime(),
		Permissions:                   permission,
		HasIssues:                     hasIssues,
		ExternalTracker:               externalTracker,
		InternalTracker:               internalTracker,
		HasWiki:                       hasWiki,
		HasProjects:                   hasProjects,
		HasReleases:                   hasReleases,
		HasPackages:                   hasPackages,
		HasActions:                    hasActions,
		ExternalWiki:                  externalWiki,
		HasPullRequests:               hasPullRequests,
		IgnoreWhitespaceConflicts:     ignoreWhitespaceConflicts,
		AllowMerge:                    allowMerge,
		AllowRebase:                   allowRebase,
		AllowRebaseMerge:              allowRebaseMerge,
		AllowSquash:                   allowSquash,
		AllowRebaseUpdate:             allowRebaseUpdate,
		DefaultDeleteBranchAfterMerge: defaultDeleteBranchAfterMerge,
		DefaultMergeStyle:             string(defaultMergeStyle),
		DefaultAllowMaintainerEdit:    defaultAllowMaintainerEdit,
		AvatarURL:                     repo.AvatarLink(ctx),
		Internal:                      !repo.IsPrivate && repo.Owner.Visibility == api.VisibleTypePrivate,
		MirrorInterval:                mirrorInterval,
		MirrorUpdated:                 mirrorUpdated,
		RepoTransfer:                  transfer,
		IsWatching:                    isWatching,
		IsStarring:                    isStarring,
	}
}

// ToRepoTransfer конвертирует models.RepoTransfer в response.RepoTransfer
func ToRepoTransfer(ctx *context.Context, t *models.RepoTransfer) *response.RepoTransfer {
	teams, _ := ToTeams(ctx, t.Teams, false)

	return &response.RepoTransfer{
		Doer:      ToUser(ctx, t.Doer, nil),
		Recipient: ToUser(ctx, t.Recipient, nil),
		Teams:     teams,
	}
}
