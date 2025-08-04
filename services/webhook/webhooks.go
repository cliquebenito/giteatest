package webhook

import (
	"fmt"
	"net/http"

	webhook2 "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/context"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/routers/api/v2/models"
)

type WebHookService struct {
}

type HookProcessor interface {
	AddRepoHook(ctx *context.APIContext, form *models.CreateHookOption) (*api.Hook, error)
}

func NewWebHookService() HookProcessor {
	return &WebHookService{}
}

// AddRepoHook adds a new hook in repo
func (w *WebHookService) AddRepoHook(ctx *context.APIContext, form *models.CreateHookOption) (*api.Hook, error) {
	hook, err := addHook(ctx, form)
	if err != nil {
		return nil, err
	}
	apiHook, ok := toAPIHook(ctx, ctx.Repo.RepoLink, hook)
	if !ok {
		return nil, err
	}
	return apiHook, nil
}

// addHook adds a new hook
func addHook(ctx *context.APIContext, form *models.CreateHookOption) (*webhook2.Webhook, error) {
	if !IsValidHookTaskType(form.Type) {
		return nil, models.ErrInvalidHookType{HookType: form.Type}
	}
	w := ToWebHookConvertor(form)
	if err := w.SetHeaderAuthorization(form.AuthorizationHeader); err != nil {
		ctx.Error(http.StatusInternalServerError, "SetHeaderAuthorization", err)
		return nil, err
	}
	if err := w.UpdateEvent(); err != nil {
		return nil, err
	} else if err := webhook2.CreateWebhook(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// toAPIHook converts webhook to api.Hook
func toAPIHook(ctx *context.APIContext, repoLink string, hook *webhook2.Webhook) (*api.Hook, bool) {
	apiHook, err := toHook(repoLink, hook)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ToHook", err)
		return nil, false
	}
	return apiHook, true
}

// toHook converts webhook to api.Hook
func toHook(repoLink string, w *webhook2.Webhook) (*api.Hook, error) {
	config := map[string]string{
		"url":          w.URL,
		"content_type": w.ContentType.Name(),
		"secret":       w.Secret,
	}

	authorizationHeader, err := w.HeaderAuthorization()
	if err != nil {
		return nil, err
	}

	return &api.Hook{
		ID:                  w.ID,
		URL:                 fmt.Sprintf("%s/settings/hooks/%d", repoLink, w.ID),
		Active:              w.IsActive,
		Config:              config,
		Events:              w.EventsArray(),
		BranchFilter:        w.BranchFilter,
		AuthorizationHeader: authorizationHeader,
		Updated:             w.UpdatedUnix.AsTime(),
		Created:             w.CreatedUnix.AsTime(),
	}, nil
}
