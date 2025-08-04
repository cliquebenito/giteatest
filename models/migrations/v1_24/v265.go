package v1_24

import (
	"code.gitea.io/gitea/models/git_hooks"
	"xorm.io/xorm"
)

// CreateScExternalHookTable создание таблицы ScExternalHook в зависимости от параметра SourceControl.Enabled и SourceControl.ExternalPreReceiveHookEnabled
func CreateScExternalHookTable(x *xorm.Engine) error {
	return x.Sync(new(git_hooks.ScGitHook))
}
