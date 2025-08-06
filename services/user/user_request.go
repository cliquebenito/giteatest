package user

import (
	"context"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/api/v2/models"
)

// CreateUserRequest usecase для создания пользователя
func CreateUserRequest(ctx context.Context, form models.CreateUserRequest) (models.CreateUserResponse, error) {
	auditParams := map[string]string{
		"email": form.Email,
	}
	u := &user.User{
		Name:               form.Name,
		FullName:           form.FullName,
		Email:              form.Email,
		MustChangePassword: true,
		LoginType:          auth.Plain,
		LoginName:          strings.ToLower(form.UserKey),
	}
	overwriteDefault := &user.CreateUserOverwriteOptions{
		IsActive: util.OptionalBoolTrue,
	}
	if form.Restricted != nil {
		overwriteDefault.IsRestricted = util.OptionalBoolOf(*form.Restricted)
	}
	if form.Visibility != "" {
		visibility := api.VisibilityModes[form.Visibility]
		overwriteDefault.Visibility = &visibility
	}
	if form.Created != nil {
		u.CreatedUnix = timeutil.TimeStamp(form.Created.Unix())
		u.UpdatedUnix = u.CreatedUnix
	}
	if err := user.CreateUser(u, overwriteDefault); err != nil {
		auditParams["error"] = "Error has occurred while creating user"
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserKey, strconv.FormatInt(u.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		if user.IsLoginNameAlreadyUsed(err) ||
			user.IsErrEmailAlreadyUsed(err) ||
			user.IsErrUserAlreadyExist(err) ||
			user.IsLoginNameAlreadyUsed(err) ||
			db.IsErrNameReserved(err) ||
			db.IsErrNameCharsNotAllowed(err) ||
			user.IsErrEmailCharIsNotSupported(err) ||
			user.IsErrEmailInvalid(err) ||
			db.IsErrNamePatternNotAllowed(err) {
			log.Error("Error has occurred while creating user: %v", err)
			return models.CreateUserResponse{}, err
		} else {
			return models.CreateUserResponse{}, err
		}
	}

	log.Debug("User created")
	audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserKey, strconv.FormatInt(u.ID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return models.CreateUserResponse{
		ID:       u.ID,
		UserKey:  u.LoginName,
		Name:     u.Name,
		FullName: u.FullName,
		Email:    u.Email,
	}, nil
}

// GetUserRequest usecase для получения пользователя
func GetUserRequest(ctx context.Context, form models.UserInfoRequest) (models.UserInfoResponse, error) {
	u, err := user.GetUserByLoginName(ctx, form.UserKey)
	if err != nil {
		log.Error("Error has occurred while getting user: %v", err)
		return models.UserInfoResponse{}, err
	}
	return models.UserInfoResponse{
		ID:         u.ID,
		UserKey:    u.LoginName,
		Name:       u.Name,
		Email:      u.Email,
		FullName:   u.FullName,
		Visibility: u.Visibility,
	}, nil
}
