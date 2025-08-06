package user_manager

import (
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

type userDB interface {
	GetUserNamesByIDs(ids []int64) ([]string, error)
}

type UserManager struct {
	db userDB
}

func NewUserManager(userDB userDB) UserManager {
	return UserManager{
		db: userDB,
	}
}

func (u UserManager) GetUserNames(ids []int64) []string {
	userNames, err := user.GetUserNamesByIDs(ids)
	if err != nil {
		log.Warn("Error has occurred while getting user names: %v. Set empty list", err)
		return []string{}
	}
	return userNames
}
