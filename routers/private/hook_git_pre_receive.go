package private

import (
	"code.gitea.io/gitea/models/git_hooks"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/private"
	"fmt"
	"net/http"
)

// GetGitHookPreReceive получение pre-receive git хука
func GetGitHookPreReceive(ctx *gitea_context.PrivateContext) {
	hook, err := git_hooks.GetGitHook(git_hooks.PreReceive)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, private.ResponseWithHook{
			Err: fmt.Sprintf("Failed to get %v hook", git_hooks.PreReceive),
		})
		return
	}

	if hook == nil {
		ctx.JSON(http.StatusOK, private.ResponseWithHook{})
		return
	}

	ctx.JSON(http.StatusOK, private.ResponseWithHook{Hook: *hook})
}
