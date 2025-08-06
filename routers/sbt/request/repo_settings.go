package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// RepoBaseSettingsOptional структура запроса обновления основных настроек репозитория
type RepoBaseSettingsOptional struct {
	RepoName    *string `json:"name" binding:"SbtAlphaDashDot;SbtMaxSize(100)"`
	Description *string `json:"description" binding:"SbtMaxSize(2048)"`
	Website     *string `json:"website" binding:"SbtUrl;SbtMaxSize(1024)"`
	Private     *bool   `json:"private"`
	Template    *bool   `json:"template"`
}

// Validate validates the fields
func (f *RepoBaseSettingsOptional) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// RepoPullsSettingsOptional структура обновления настроек стилей слияния ПР-ов
type RepoPullsSettingsOptional struct {
	EnablePulls                   *bool   `json:"enablePulls"`
	PullsIgnoreWhitespace         *bool   `json:"pullsIgnoreWhitespace"`
	PullsAllowMerge               *bool   `json:"pullsAllowMerge"`
	PullsAllowRebase              *bool   `json:"pullsAllowRebase"`
	PullsAllowRebaseMerge         *bool   `json:"pullsAllowRebaseMerge"`
	PullsAllowSquash              *bool   `json:"pullsAllowSquash"`
	PullsAllowManualMerge         *bool   `json:"pullsAllowManualMerge"`
	PullsDefaultMergeStyle        *string `json:"pullsDefaultMergeStyle" binding:"SbtIn(merge,rebase,rebase-merge,squash)"`
	EnableAutodetectManualMerge   *bool   `json:"enableAutodetectManualMerge"`
	PullsAllowRebaseUpdate        *bool   `json:"pullsAllowRebaseUpdate"`
	DefaultDeleteBranchAfterMerge *bool   `json:"defaultDeleteBranchAfterMerge"`
	DefaultAllowMaintainerEdit    *bool   `json:"defaultAllowMaintainerEdit"`
}

// Validate validates the fields
func (f *RepoPullsSettingsOptional) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// TransferRepoOptional структура запроса на передачу прав на репозиторий другому пользователю
type TransferRepoOptional struct {
	NewOwnerName string `json:"newOwnerName" binding:"Required"`
}

// Validate validates the fields
func (f *TransferRepoOptional) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
