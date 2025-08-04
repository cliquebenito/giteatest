package v1_25

import (
	"xorm.io/xorm"
)

// AddEnableSonarQube добавляем новое поле EnableSonarQube в таблицу ProtectedBranch
func AddEnableSonarQube(x *xorm.Engine) error {
	type ProtectedBranch struct {
		EnableSonarQube bool `xorm:"NOT NULL DEFAULT false"`
	}
	return x.Sync(new(ProtectedBranch))
}
