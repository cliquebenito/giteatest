package models

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/api/v2/models/metrics"
	"code.gitea.io/gitea/routers/api/v2/models/repo"
	"code.gitea.io/gitea/routers/api/v2/models/ssh"
)

// ParseRepoGetOpts парсит из запроса опции для запроса репозитория
func ParseRepoGetOpts(ctx *context.APIContext) *repo.RepositoryGetOptions {
	return &repo.RepositoryGetOptions{
		RepoKey:    ctx.FormString("repo_key"),
		TenantKey:  ctx.FormString("tenant_key"),
		ProjectKey: ctx.FormString("project_key"),
	}
}

func ParseInternalMetricGetOpts(ctx *context.APIContext) *metrics.InternalMetricGetOptions {
	return &metrics.InternalMetricGetOptions{
		RepoKey:    ctx.FormString("repo_key"),
		TenantKey:  ctx.FormString("tenant_key"),
		ProjectKey: ctx.FormString("project_key"),
		Metric:     ctx.FormString("metric"),
	}
}

func ParseExternalMetricGetOpts(ctx *context.APIContext) *metrics.ExternalMetricOptions {
	return &metrics.ExternalMetricOptions{
		RepoKey:    ctx.FormString("repo_key"),
		TenantKey:  ctx.FormString("tenant_key"),
		ProjectKey: ctx.FormString("project_key"),
	}
}

func ParseUserKeyGetOpt(ctx *context.APIContext) string {
	return ctx.FormString("user_key")
}

func ParseSSHKeyGetOpts(ctx *context.APIContext) ssh.SSHKeyOptions {
	return ssh.SSHKeyOptions{
		UserKey: ctx.FormString("user_key"),
		Title:   ctx.FormString("title"),
	}
}
