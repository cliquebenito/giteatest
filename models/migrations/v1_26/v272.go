package v1_26

import (
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/models/unit_links_sender"
)

// CreateUnitLinks создаем таблицу для хранения информации o связях юнитов в SourceControl и TaskTracker
func CreateUnitLinks(x *xorm.Engine) error {
	return x.Sync(new(unit_links.UnitLinks))
}

// CreateUnitLinksSenderTasks создаем таблицу для хранения информации об отправке связей юнитов в TaskTracker
func CreateUnitLinksSenderTasks(x *xorm.Engine) error {
	return x.Sync(new(unit_links_sender.UnitLinksSenderTasks))
}
