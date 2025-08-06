package createuser

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/db"
	usermodel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

func Create(ctx context.Context, engine db.Engine, user *usermodel.User) error {
	auditParams := map[string]string{}

	txCtx, committer, err := db.TxContext(ctx)
	if err != nil {
		auditParams["error"] = "Error has occurred while receiving the transaction parameters"
		audit.CreateAndSendEvent(audit.UserCreateEvent, audit.EmptyRequiredField, strconv.FormatInt(user.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("create tx: %w", err)
	}

	defer func() {
		if err = committer.Close(); err != nil {
			auditParams["error"] = "Error has occurred while closing the transaction"
			audit.CreateAndSendEvent(audit.UserCreateEvent, audit.EmptyRequiredField, strconv.FormatInt(user.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			log.Error("close tx: %s", err.Error())
		}
	}()

	if err = createUser(txCtx, engine, user); err != nil {
		auditParams["error"] = "Error has occurred while creating user"
		audit.CreateAndSendEvent(audit.UserCreateEvent, audit.EmptyRequiredField, strconv.FormatInt(user.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("create user: %w", err)
	}

	if err = committer.Commit(); err != nil {
		log.Error("Error has occurred while commiting %v", err)
		return fmt.Errorf("commit: %w", err)
	}

	newValueBytes, err := json.Marshal(struct{ ID int64 }{ID: user.ID})
	if err != nil {
		log.Error("Error has occurred while marshalling %v", err)
		return fmt.Errorf("marshal: %w", err)
	}

	auditParams["new_value"] = string(newValueBytes)
	audit.CreateAndSendEvent(audit.UserCreateEvent, audit.EmptyRequiredField, strconv.FormatInt(user.ID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)

	log.Debug("Account created: %s", user.Name)

	return nil
}

func createUser(ctx context.Context, _ db.Engine, user *usermodel.User) error {
	if user == nil {
		return fmt.Errorf("empty user")
	}

	if err := validateUser(ctx, user); err != nil {
		return fmt.Errorf("invalid user: %w", err)
	}

	setDefaults(user)

	user.LowerName = strings.ToLower(user.Name)
	user.AvatarEmail = user.Email

	var err error

	if user.CreatedUnix == 0 {
		// Caller expects auto-time for creation & update timestamps.
		err = db.Insert(ctx, user)
	} else {
		// Caller sets the timestamps themselves. They are responsible for ensuring
		// both `CreatedUnix` and `UpdatedUnix` are set appropriately.
		_, err = db.GetEngine(ctx).NoAutoTime().Insert(user)
	}

	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	// Не создаем запись об email, если адрес пустой
	if user.Email != "" {
		emailToCreate := &usermodel.EmailAddress{
			IsPrimary:   true,
			UID:         user.ID,
			Email:       user.Email,
			IsActivated: user.IsActive,
			LowerEmail:  strings.ToLower(user.Email),
		}

		if err = db.Insert(ctx, emailToCreate); err != nil {
			return err
		}
	}

	count, err := usermodel.CountUsersCtx(ctx, nil)
	if err != nil {
		log.Error("Error has occurred while getting count users %v", err)
		return fmt.Errorf("count users: %w", err)
	}

	if count == 1 {
		user.IsAdmin = true
		user.IsActive = true
		user.SetLastLogin()

		if err = usermodel.UpdateUserCols(ctx, user, "is_admin", "is_active", "last_login_unix"); err != nil {
			log.Error("Error has occurred while creating first user as admin %v", err)
			return fmt.Errorf("create first user as admin: %w", err)
		}
	}

	return nil
}

func setDefaults(user *usermodel.User) {
	user.KeepEmailPrivate = setting.Service.DefaultKeepEmailPrivate
	user.Visibility = setting.Service.DefaultUserVisibilityMode
	user.AllowCreateOrganization = setting.Service.DefaultAllowCreateOrganization && !setting.Admin.DisableRegularOrgCreation
	user.EmailNotificationsPreference = setting.Admin.DefaultEmailNotification
	user.MaxRepoCreation = -1
	user.Theme = setting.UI.DefaultTheme
	user.IsRestricted = setting.Service.DefaultUserIsRestricted
	user.IsActive = !(setting.Service.RegisterEmailConfirm || setting.Service.RegisterManualConfirm)

	if user.UpdatedUnix < user.CreatedUnix {
		user.UpdatedUnix = user.CreatedUnix
	}
}

func validateUser(ctx context.Context, user *usermodel.User) error {
	if err := usermodel.IsUsableUsername(user.Name); err != nil {
		return fmt.Errorf("username %q is invalid: %v", user.Name, err)
	}

	isExists, err := usermodel.IsUserExist(ctx, 0, user.Name)
	if err != nil {
		return fmt.Errorf("check if user exists: %w", err)
	}
	if isExists {
		return usermodel.ErrUserAlreadyExist{Name: user.Name}
	}

	isExists, err = usermodel.IsEmailUsed(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("check if email used: %w", err)
	}
	if isExists && user.Email != "" {
		return usermodel.ErrEmailAlreadyUsed{Email: user.Email}
	}

	return nil
}
