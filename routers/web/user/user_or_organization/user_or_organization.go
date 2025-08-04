package user_or_organization

import (
	"net/http"
	"strings"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/web/feed"
	"code.gitea.io/gitea/routers/web/user"
	context_service "code.gitea.io/gitea/services/context"
)

func (s Server) GetUserOrOrganizationSubRoute(ctx *context.Context) {
	// WORKAROUND to support usernames with "." in it
	// https://github.com/go-chi/chi/issues/781

	username := ctx.Params("username")

	reloadParam := func(suffix string) (success bool) {
		ctx.SetParams("username", strings.TrimSuffix(username, suffix))
		context_service.UserAssignmentWeb()(ctx)
		return !ctx.Written()
	}

	switch {
	case strings.HasSuffix(username, ".png"):
		if reloadParam(".png") {
			user.AvatarByUserName(ctx)
		}
	case strings.HasSuffix(username, ".keys"):
		if reloadParam(".keys") {
			user.ShowSSHKeys(ctx)
		}
	case strings.HasSuffix(username, ".gpg"):
		if reloadParam(".gpg") {
			user.ShowGPGKeys(ctx)
		}
	case strings.HasSuffix(username, ".rss"):
		if !setting.Other.EnableFeed {
			ctx.Error(http.StatusNotFound)
			return
		}
		if reloadParam(".rss") {
			context_service.UserAssignmentWeb()(ctx)
			feed.ShowUserFeedRSS(ctx)
		}
	case strings.HasSuffix(username, ".atom"):
		if !setting.Other.EnableFeed {
			ctx.Error(http.StatusNotFound)
			return
		}
		if reloadParam(".atom") {
			feed.ShowUserFeedAtom(ctx)
		}
	default:
		s.handleUserOrOrganizationRequest(ctx)
	}
}
