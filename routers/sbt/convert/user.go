package convert

import (
	"code.gitea.io/gitea/models/perm"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/routers/sbt/response"

	"context"
)

// ToUser конвертирует user_model.User в response.User
// если Doer известен, добавляется частная информация доступная ему
func ToUser(ctx context.Context, user, doer *userModel.User) *response.User {
	if user == nil {
		return nil
	}
	authed := false
	signed := false
	if doer != nil {
		signed = true
		authed = doer.ID == user.ID || doer.IsAdmin
	}
	return toUser(ctx, user, signed, authed)
}

// ToUserWithAccessMode конвертирует userModel.User в response.User с учетом perm.AccessMode
func ToUserWithAccessMode(ctx context.Context, user *userModel.User, accessMode perm.AccessMode) *response.User {
	if user == nil {
		return nil
	}
	return toUser(ctx, user, accessMode != perm.AccessModeNone, false)
}

// toUser конвертирует userModel.User в response.User
func toUser(ctx context.Context, user *userModel.User, signed, authed bool) *response.User {
	result := &response.User{
		ID:          user.ID,
		UserName:    user.Name,
		FullName:    user.FullName,
		Email:       user.GetEmail(),
		AvatarURL:   user.AvatarLink(ctx),
		Created:     user.CreatedUnix.AsTime(),
		Restricted:  user.IsRestricted,
		Location:    user.Location,
		Website:     user.Website,
		Description: user.Description,
		// счетчики
		Followers:    user.NumFollowers,
		Following:    user.NumFollowing,
		StarredRepos: user.NumStars,
	}

	result.Visibility = user.Visibility.String()

	// прячет email  в зависимости от настроек видимости
	if signed && (!user.KeepEmailPrivate || authed) {
		result.Email = user.Email
	}

	if authed {
		result.IsAdmin = user.IsAdmin
		result.LoginName = user.LoginName
		result.LastLogin = user.LastLoginUnix.AsTime()
		result.Language = user.Language
		result.IsActive = user.IsActive
		result.ProhibitLogin = user.ProhibitLogin
	}
	return result
}
