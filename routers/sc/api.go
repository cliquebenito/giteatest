package sc

import (
	gocontext "context"
	"net/http"

	"github.com/go-chi/cors"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/role_model"
	context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/sc/configurations/controller"
	"code.gitea.io/gitea/routers/sc/configurations/usecase"
	"code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/auth/iamprivileger"
)

func Routes() *web.Route {
	ctx := gocontext.Background()
	m := web.NewRoute()
	engine := db.GetEngine(ctx)
	_ = engine
	m.Use(securityHeaders())
	if setting.CORSConfig.Enabled {
		m.Use(cors.Handler(cors.Options{
			AllowedOrigins:   setting.CORSConfig.AllowDomain,
			AllowedMethods:   setting.CORSConfig.Methods,
			AllowCredentials: setting.CORSConfig.AllowCredentials,
			AllowedHeaders:   append([]string{"Authorization", "X-SourceControl-OTP"}, setting.CORSConfig.Headers...),
			MaxAge:           int(setting.CORSConfig.MaxAge.Seconds()),
		}))
	}
	m.Use(context.APIContexter())
	group := buildAuthGroup()
	if err := group.Init(ctx); err != nil {
		log.Error("Error has occurred while initializing '%s' auth method. Error: %v", group.Name(), err)
		return nil
	}

	m.Use(auth.APIAuth(group))

	m.Use(auth.VerifyAuthWithOptionsAPI(&auth.VerifyOptions{
		SignInRequired: setting.Service.RequireSignInView,
	}))
	// TODO подумать над сваггером
	m.Group("", func() {
		m.Get("/swagger", func(ctx *context.APIContext) {
			ctx.Redirect(setting.AppSubURL + "/api/swagger_ui")
		})
	})

	uc := usecase.NewUsecase()
	cfg := controller.NewScConfigurations(uc)

	m.Post("/configurations", isSigned(), cfg.Configurations)

	return m
}

func buildAuthGroup() *auth.Group {
	group := auth.NewGroup(
		&auth.OAuth2{},
		&auth.HTTPSign{},
		&auth.Basic{},
	)

	enforcer := role_model.GetSecurityEnforcer()
	engine := db.GetEngine(gocontext.Background())

	privileger, err := iamprivileger.New(enforcer, engine)
	if err != nil {
		log.Fatal("create privileger: %s", err)
	}

	//if setting.IAM.Enabled {
	group.Add(auth.NewIAMProxy(privileger, engine))
	//}

	return group
}
func isSigned() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		if !ctx.IsSigned {
			ctx.Error(http.StatusUnauthorized, "Assign", "Err: need authorization")
			return
		}
	}
}
func securityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			resp.Header().Set("x-content-type-options", "nosniff")
			next.ServeHTTP(resp, req)
		})
	}
}
