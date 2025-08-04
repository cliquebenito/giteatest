package git_hooks

import (
	"code.gitea.io/gitea/models/db"
	"time"
	"xorm.io/builder"
)

func init() {
	db.RegisterModel(new(ScGitHook))
}

// ScGitHook таблица для хранения git хуков
type ScGitHook struct {
	ID                   int64         `xorm:"pk autoincr"`
	Path                 string        `xorm:"not null"`
	HookType             ScGitHookType `xorm:"index not null"`
	Timeout              time.Duration
	PositionalParameters map[string]string
}

// InsertOrUpdateGitHook добавление или изменение git хуков (если git хуков не было то они добавляются, если были то обновляются новыми значениями)
func InsertOrUpdateGitHook(path string, hookType ScGitHookType, timeout int64, positionalParameters map[string]string) (old *ScGitHook, err error) {
	old = &ScGitHook{}
	has, err := db.GetEngine(db.DefaultContext).Where(builder.Eq{"hook_type": hookType}).Get(old)
	if err != nil {
		return old, err
	} else if !has {
		return old, db.Insert(db.DefaultContext, ScGitHook{
			Path:                 path,
			HookType:             hookType,
			Timeout:              time.Duration(timeout) * time.Millisecond,
			PositionalParameters: positionalParameters,
		})
	} else {
		_, err = db.GetEngine(db.DefaultContext).ID(old.ID).Update(ScGitHook{
			Path:                 path,
			HookType:             hookType,
			Timeout:              time.Duration(timeout) * time.Millisecond,
			PositionalParameters: positionalParameters,
		})
	}
	return old, err
}

// GetGitHook Получение git хука по его типу
func GetGitHook(hookType ScGitHookType) (*ScGitHook, error) {
	var res ScGitHook
	has, err := db.GetEngine(db.DefaultContext).Where(builder.Eq{"hook_type": hookType}).Get(&res)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return &res, nil
}

// DeleteGitHook Удаление git хука по его типу
func DeleteGitHook(hookType ScGitHookType) error {
	_, err := db.GetEngine(db.DefaultContext).Delete(&ScGitHook{HookType: hookType})

	return err
}
