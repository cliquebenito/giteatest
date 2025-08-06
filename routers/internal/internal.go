package internal

import (
	"net/http"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/routers/web/healthcheck"
	role_model_router "code.gitea.io/gitea/routers/web/role_model"
	"code.gitea.io/gitea/services/forms"
)

// Routes registers all internal APIs routes to web application.
// These APIs will be invoked by internal commands for example `gitea serv` and etc.
func Routes() *web.Route {
	r := web.NewRoute()

	var mid []any
	mid = append(mid, common.Sessioner(), context.Contexter())
	r.Use(mid...)

	registerRoutes(r)

	return r
}

// registerRoutes register routes
func registerRoutes(m *web.Route) {

	reqMultiTenantEnabled := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			if !setting.SourceControl.MultiTenantEnabled {
				ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
			}
		}
	}

	m.Group("/tenants", func() {
		m.Get("", role_model_router.GetDefaultTenant)
		m.Post("/create", web.Bind(forms.CreateTenantApiForm{}), role_model_router.CreateTenant)
		m.Group("/{tenantid}", func() {
			m.Get("", role_model_router.GetTenantByID)
			m.Patch("/edit", web.Bind(forms.EditTenantApiForm{}), role_model_router.EditTenant)
			m.Post("/activate", role_model_router.ActivateTenant)
			m.Post("/deactivate", role_model_router.DeactivateTenant)
			m.Delete("/delete", role_model_router.DeleteTenant)
		})
	}, reqMultiTenantEnabled())

	m.Group("/projects", func() {
		m.Get("/{projectid}", role_model_router.GetProjectByID)
		m.Post("/create", web.Bind(forms.CreateProjectApiForm{}), role_model_router.CreateProject)
		m.Patch("/edit", web.Bind(forms.ModifyProjectApiForm{}), role_model_router.EditProject)
		m.Delete("/delete", web.Bind(forms.DeleteProjectApiForm{}), role_model_router.DeleteProject)
	})

	m.Group("/one_work", func() {
		m.Get("/api/healthz", healthcheck.Check)
	})
}
