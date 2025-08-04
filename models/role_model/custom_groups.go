package role_model

import (
	"context"
	"fmt"
	"time"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
)

// ScCustomPrivilegesGroup - структура таблицы sc_custom_privileges_group
type ScCustomPrivilegesGroup struct {
	Id         int64     `xorm:"pk autoincr"`
	Code       string    `xorm:"VARCHAR(100)"`
	Name       string    `xorm:"VARCHAR(255)"`
	Privileges string    `xorm:"VARCHAR(255)"`
	Created    time.Time `xorm:"created"`
	Updated    time.Time `xorm:"updated"`
}

func NewScCustomPrivilegesGroup(code, name, privileges string) *ScCustomPrivilegesGroup {
	return &ScCustomPrivilegesGroup{
		Code:       code,
		Name:       name,
		Privileges: privileges,
	}
}
func init() {
	db.RegisterModel(new(ScCustomPrivilegesGroup))
}

// GetAllCustomPrivilegesGroup - возвращает все группы привилегий из таблицы sc_custom_privileges_group
func GetAllCustomPrivilegesGroup(ctx context.Context) ([]*ScCustomPrivilegesGroup, error) {
	allGroups := make([]*ScCustomPrivilegesGroup, 0)
	err := db.GetEngine(ctx).Find(&allGroups)
	if err != nil {
		return nil, fmt.Errorf("error has occurred while getting all custom groups: %w", err)
	}
	return allGroups, nil
}

// GetCustomPrivilegesGroupByCode - возвращает группу привилегий из таблицы sc_custom_privileges_group по коду
func GetCustomPrivilegesGroupByCode(ctx context.Context, groupCode string) (*ScCustomPrivilegesGroup, error) {
	group := &ScCustomPrivilegesGroup{Code: groupCode}
	has, err := db.GetEngine(ctx).Get(group)
	if err != nil {
		return nil, fmt.Errorf("error has occurred while getting custom group by code: %w", err)
	}
	if !has {
		return nil, &ErrCustomGroupNotFound{Group: groupCode}
	}

	return group, nil
}

// AddCustomPrivilegesGroup - добавляет группу привилегий в таблицу sc_custom_privileges_group
func AddCustomPrivilegesGroup(ctx context.Context, group *ScCustomPrivilegesGroup) error {
	_, err := db.GetEngine(ctx).Insert(group)
	if err != nil {
		return fmt.Errorf("error has occurred while adding custom group: %w", err)
	}
	return nil
}

// UpdateCustomPrivilegesGroupByCode - обновляет группу привилегий в таблице sc_custom_privileges_group
func UpdateCustomPrivilegesGroupByCode(ctx context.Context, group *ScCustomPrivilegesGroup) error {
	_, err := db.GetEngine(ctx).Where(builder.Eq{"code": group.Code}).Update(group)
	if err != nil {
		return fmt.Errorf("error has occurred while updating custom group: %w", err)
	}
	return nil
}

// DeleteCustomPrivilegesGroupByCode - удаляет группу привилегий из таблицы sc_custom_privileges_group
func DeleteCustomPrivilegesGroupByCode(ctx context.Context, code string) error {
	_, err := db.GetEngine(ctx).Delete(&ScCustomPrivilegesGroup{Code: code})
	if err != nil {
		return fmt.Errorf("error has occurred while deleting custom group: %w", err)
	}
	return nil
}
