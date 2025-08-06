package user

import (
	"context"
	"strings"

	"code.gitea.io/gitea/models/db"
)

func GetIAMUserByLoginName(ctx context.Context, engine db.Engine, name string) (*User, error) {
	if len(name) == 0 {
		return nil, &ErrUserNotExist{0, name, 0}
	}
	u := &User{LoginName: strings.ToLower(name)}

	has, err := engine.Get(u)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, &ErrUserNotExist{0, name, 0}
	}

	return u, nil
}
