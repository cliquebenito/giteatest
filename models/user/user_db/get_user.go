package user_db

import "code.gitea.io/gitea/models/user"

func (u UserDB) GetUserNamesByIDs(ids []int64) ([]string, error) {
	return user.GetUserNamesByIDs(ids)
}
