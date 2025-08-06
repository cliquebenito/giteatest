package admin

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"code.gitea.io/gitea/models/git_hooks"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	audit_utils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
)

const (
	tplGitHooks = "admin/git_hook"
)

// PreReceiveHook Получение pre-receive git хука
func PreReceiveHook(ctx *context.Context) {
	ctx.Data["PageIsAdminGitHooks"] = true

	hook, err := git_hooks.GetGitHook(git_hooks.PreReceive)
	if err != nil {
		log.Error("Error has occurred while getting git pre-receive hook, error: %v", err)
		ctx.Error(http.StatusInternalServerError)
		return
	}

	ctx.Data["PreReceiveHook"] = hook
	if hook != nil {
		ctx.Data["PreReceiveHookTimeoutMs"] = hook.Timeout.Milliseconds()
	}

	ctx.HTML(http.StatusOK, tplGitHooks)
}

// PreReceiveHookPost Создание или обновление pre-receive git хука
func PreReceiveHookPost(ctx *context.Context) {
	ctx.Data["PageIsAdminGitHooks"] = true
	form := web.GetForm(ctx).(*forms.PreReceiveHookForm)

	type auditValue struct {
		Path       string
		Timeout    int64
		Parameters map[string]string
	}

	newValue := auditValue{
		Path:       form.Path,
		Timeout:    form.Timeout,
		Parameters: form.Parameters,
	}

	auditParams := make(map[string]string, 0)

	auditValues := audit_utils.NewRequiredAuditParams(ctx)

	event := audit.GitHookAddEvent
	if form.Timeout < 0 {
		log.Error("error has occurred while trying to insert or update git pre-receive hook: negative timeout was passed")
		auditParams["error"] = "Error has occurred while trying to insert or update git pre-receive hook: negative timeout was passed"
		audit.CreateAndSendEvent(event, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Нельзя установить отрицательный таймаут для git-хука")
		return
	}

	newValueBytes, jsonMarshalErr := json.Marshal(newValue)
	if jsonMarshalErr != nil {
		log.Error("error has occurred while converting data to bytes: %v, error: %v", newValue, jsonMarshalErr)
		ctx.Error(http.StatusBadRequest, "Не удалось преобразовать данные")
		auditParams["error"] = "Error has occurred while converting data to bytes"
		audit.CreateAndSendEvent(event, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
	}
	auditParams["new_value"] = string(newValueBytes)
	if ctx.Written() {
		return
	}

	path := form.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(setting.AppWorkPath, path)
	}

	_, err := os.Stat(path)
	if err == nil {
		old, err := git_hooks.InsertOrUpdateGitHook(path, git_hooks.PreReceive, form.Timeout, form.Parameters)
		if old != nil && old.ID != 0 {
			event = audit.GitHookEditEvent
			oldValueBytes, _ := json.Marshal(old)
			auditParams["old_value"] = string(oldValueBytes)
		}
		if err != nil {
			log.Error("Error has occurred while try insert or update git pre-receive hook, error: %v", err)
			auditParams["error"] = "Error has occurred while try insert or update git pre-receive hook"
			audit.CreateAndSendEvent(event, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError)
			return
		}
	}
	if os.IsNotExist(err) {
		log.Debug("File %v not exists, error: %v", path, err)
		auditParams["error"] = "File not exists"
		audit.CreateAndSendEvent(event, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, ctx.Tr("admin.file_not_exist"))
		return
	}

	audit.CreateAndSendEvent(event, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusOK)
}

// PreReceiveHookDelete Удаление pre-receive git хука
func PreReceiveHookDelete(ctx *context.Context) {
	ctx.Data["PageIsAdminGitHooks"] = true
	auditParams := make(map[string]string)

	err := git_hooks.DeleteGitHook(git_hooks.PreReceive)
	if err != nil {
		auditParams["error"] = "Error has occurred while try delete pre-receive hook"
		log.Error("Error has occurred while try delete pre-receive hook, error: %v", err)
		audit.CreateAndSendEvent(audit.GitHookRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError)
		return
	}

	audit.CreateAndSendEvent(audit.GitHookRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Status(http.StatusNoContent)
}
