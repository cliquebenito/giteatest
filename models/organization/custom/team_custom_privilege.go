package custom

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// //go:generate mockery --name=dbEngine --exported
type dbEngine interface {
	Get(beans ...interface{}) (bool, error)
	Insert(beans ...interface{}) (int64, error)
	Where(interface{}, ...interface{}) *xorm.Session
	Find(interface{}, ...interface{}) error
	Delete(...interface{}) (int64, error)
	Context(ctx context.Context) *xorm.Session
}

type customPrivilegeDB struct {
	engine dbEngine
}

type CustomPrivileger interface {
	InsertCustomPrivilegesForTeam(teamCustomPrivilege []ScTeamCustomPrivilege) error
	GetCustomPrivilegesByTeam(teamName string) ([]*ScTeamCustomPrivilege, error)
	DeleteCustomPrivilegesByTeam(teamName string) error
	GetCustomPrivilegesByBranchAndRepoID(ctx context.Context, branchName string, repoID int64) ([]ScTeamCustomPrivilege, error)
	DeleteCustomPrivilegesByParams(ctx context.Context, teamCustomPrivilege ScTeamCustomPrivilege) error
}

func NewCustomDB(engine dbEngine) customPrivilegeDB {
	return customPrivilegeDB{engine: engine}
}

func init() {
	db.RegisterModel(new(ScTeamCustomPrivilege))
}

// ScTeamCustomPrivilege таблица для хранения полей кастомных привилегий для команды
type ScTeamCustomPrivilege struct {
	ID               int64  `xorm:"PK AUTOINCR"`
	TeamName         string `xorm:"VARCHAR(255)"`
	RepositoryID     int64
	AllRepositories  bool   `xorm:"NOT NULL DEFAULT false"`
	CustomPrivileges string `xorm:"VARCHAR(20)"`
}

// InsertCustomPrivilegesForTeam вставляем информацию о кастомных привилегиях для команды
func (c customPrivilegeDB) InsertCustomPrivilegesForTeam(teamCustomPrivilege []ScTeamCustomPrivilege) error {
	if _, err := c.engine.Insert(teamCustomPrivilege); err != nil {
		log.Error("Error has occurred while inserting custom privileges err: %v", err)
		return fmt.Errorf("insert custom privileges: %w", err)
	}
	return nil
}

// GetCustomPrivilegesByTeam получаем информацию о кастомных привилегиях по команде
func (c customPrivilegeDB) GetCustomPrivilegesByTeam(teamName string) ([]*ScTeamCustomPrivilege, error) {
	var customPrivileges []*ScTeamCustomPrivilege

	if err := c.engine.Where(builder.Eq{"team_name": teamName}).Find(&customPrivileges); err != nil {
		log.Error("Error has occurred while getting custom privileges for team %s err: %v", teamName, err)
		return nil, fmt.Errorf("get custom privileges: %w", err)
	}
	return customPrivileges, nil
}

// DeleteCustomPrivilegesByTeam удаляем информацию о кастомных привилегиях по команде
func (c customPrivilegeDB) DeleteCustomPrivilegesByTeam(teamName string) error {
	if _, err := c.engine.Delete(&ScTeamCustomPrivilege{TeamName: teamName}); err != nil {
		log.Error("Error has occurred while deleting for team: %s err: %v", teamName, err)
		return fmt.Errorf("delete custom privileges for team: %w", err)
	}
	return nil
}

// GetCustomPrivilegesByBranchAndRepoID получаем список кастомных привилегий для команды для отображения на странице команды
func (c customPrivilegeDB) GetCustomPrivilegesByBranchAndRepoID(ctx context.Context, branchName string, repoID int64) ([]ScTeamCustomPrivilege, error) {
	var customPrivileges []ScTeamCustomPrivilege

	sess := c.engine.Where(
		builder.And(
			builder.Eq{"all_repositories": true},
		)).
		Or(builder.And(
			builder.Eq{"repository_id": repoID},
		))
	if branchName == "" {
		sess.Or(builder.And(
			builder.Eq{"repository_id": repoID},
			builder.Eq{"all_repositories": false},
		))
	} else {
		sess.Or(builder.And(
			builder.Eq{"branch_name": branchName},
			builder.Eq{"repository_id": repoID},
		))
	}

	if err := sess.Find(&customPrivileges); err != nil {
		log.Error("Error has occurred while deleting for branch: %s, repo_id: %d err: %v", branchName, repoID, err)
		return nil, fmt.Errorf("get custom privileges for bracnh: %w", err)
	}
	return customPrivileges, nil
}

// DeleteCustomPrivilegesByParams удаляем запись о кастомной привилегии у пользователя
func (c customPrivilegeDB) DeleteCustomPrivilegesByParams(ctx context.Context, teamCustomPrivilege ScTeamCustomPrivilege) error {
	if _, err := c.engine.Delete(&teamCustomPrivilege); err != nil {
		log.Error("Error has occurred while deleting custom privileges err: %v", err)
		return fmt.Errorf("delete custom privileges: %w", err)
	}
	return nil
}
