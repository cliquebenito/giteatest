package repo

import (
	repoModel "code.gitea.io/gitea/models/repo"
	unitModel "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"net/http"
)

// UpdateRepoPullsSettings Метод обновления настроек способов слияния пулл реквестов
func UpdateRepoPullsSettings(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.RepoPullsSettingsOptional)
	repo := ctx.Repo.Repository

	var units []repoModel.RepoUnit
	var deleteUnitTypes []unitModel.Type

	// Достаем настройки слияния пулл реквестов для репозитория (err возвращается только если нет unit: эту вероятность обрабатываем)
	unit, _ := repo.GetUnit(ctx, unitModel.TypePullRequests)

	if ((req.EnablePulls != nil && *req.EnablePulls) || (unit != nil && req.EnablePulls == nil)) &&
		!unitModel.TypePullRequests.UnitGlobalDisabled() {
		config := &repoModel.PullRequestsConfig{}
		if unit != nil {
			config = unit.PullRequestsConfig()
		}
		if req.PullsIgnoreWhitespace != nil {
			config.IgnoreWhitespaceConflicts = *req.PullsIgnoreWhitespace
		}
		if req.PullsAllowMerge != nil {
			config.AllowMerge = *req.PullsAllowMerge
		}
		if req.PullsAllowRebase != nil {
			config.AllowRebase = *req.PullsAllowRebase
		}
		if req.PullsAllowRebaseMerge != nil {
			config.AllowRebaseMerge = *req.PullsAllowRebaseMerge
		}
		if req.PullsAllowSquash != nil {
			config.AllowSquash = *req.PullsAllowSquash
		}
		if req.PullsAllowManualMerge != nil {
			config.AllowManualMerge = *req.PullsAllowManualMerge
		}
		if req.EnableAutodetectManualMerge != nil {
			config.AutodetectManualMerge = *req.EnableAutodetectManualMerge
		}
		if req.PullsAllowRebaseUpdate != nil {
			config.AllowRebaseUpdate = *req.PullsAllowRebaseUpdate
		}
		if req.DefaultDeleteBranchAfterMerge != nil {
			config.DefaultDeleteBranchAfterMerge = *req.DefaultDeleteBranchAfterMerge
		}
		if req.PullsDefaultMergeStyle != nil {
			config.DefaultMergeStyle = repoModel.MergeStyle(*req.PullsDefaultMergeStyle)
		}
		if req.DefaultAllowMaintainerEdit != nil {
			config.DefaultAllowMaintainerEdit = *req.DefaultAllowMaintainerEdit
		}

		units = append(units, repoModel.RepoUnit{
			RepoID: repo.ID,
			Type:   unitModel.TypePullRequests,
			Config: config,
		})
	} else if !unitModel.TypePullRequests.UnitGlobalDisabled() {
		deleteUnitTypes = append(deleteUnitTypes, unitModel.TypePullRequests)
	}

	if err := repoModel.UpdateRepositoryUnits(repo, units, deleteUnitTypes); err != nil {
		log.Error("An error has occurred while updating pulls settings for repoId: %d, error: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	ctx.Status(http.StatusOK)
}
