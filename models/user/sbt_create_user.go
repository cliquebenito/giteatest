package user

import (
	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/setting"
	"strings"
)

// SbtCreateUser сохранение пользователя. Метод написан по подобию оригинального метода CreateUser
func SbtCreateUser(u *User) (err error) {
	//Проверка на зарезервированные имена
	if err = IsUsableUsername(u.Name); err != nil {
		return err
	}

	// Устанавливаем пользователю параметры по дефолту
	u.KeepEmailPrivate = setting.Service.DefaultKeepEmailPrivate
	u.Visibility = setting.Service.DefaultUserVisibilityMode
	u.AllowCreateOrganization = setting.Service.DefaultAllowCreateOrganization && !setting.Admin.DisableRegularOrgCreation
	u.EmailNotificationsPreference = setting.Admin.DefaultEmailNotification
	u.MaxRepoCreation = -1
	u.Theme = setting.UI.DefaultTheme
	u.IsRestricted = setting.Service.DefaultUserIsRestricted
	u.IsActive = !(setting.Service.RegisterEmailConfirm || setting.Service.RegisterManualConfirm)

	// Ensure consistency of the dates.
	if u.UpdatedUnix < u.CreatedUnix {
		u.UpdatedUnix = u.CreatedUnix
	}

	// validate data
	if err := validateUser(u); err != nil {
		return err
	}

	if err := ValidateEmail(u.Email); err != nil {
		return err
	}

	//открытие транзакционного контекста
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	isExist, err := IsUserExist(ctx, 0, u.Name)
	if err != nil {
		return err
	} else if isExist {
		return ErrUserAlreadyExist{u.Name}
	}

	isExist, err = IsEmailUsed(ctx, u.Email)
	if err != nil {
		return err
	} else if isExist {
		return ErrEmailAlreadyUsed{
			Email: u.Email,
		}
	}

	u.LowerName = strings.ToLower(u.Name)
	u.AvatarEmail = u.Email
	if u.Rands, err = GetUserSalt(); err != nil {
		return err
	}
	password := u.Passwd

	if setting.SbtKeycloakForm.Enabled {
		u.LoginType = auth.OAuth2
		u.Passwd = ""
	} else {
		if err = u.SetPassword(u.Passwd); err != nil {
			return err
		}
	}
	// save changes to database
	if err = DeleteUserRedirect(ctx, u.Name); err != nil {
		return err
	}

	if u.CreatedUnix == 0 {
		// Caller expects auto-time for creation & update timestamps.
		err = db.Insert(ctx, u)
	} else {
		// Caller sets the timestamps themselves. They are responsible for ensuring
		// both `CreatedUnix` and `UpdatedUnix` are set appropriately.
		_, err = db.GetEngine(ctx).NoAutoTime().Insert(u)
	}
	if err != nil {
		return err
	}

	// Сохранение адреса электронной почты в таблице email
	if err := db.Insert(ctx, &EmailAddress{
		UID:         u.ID,
		Email:       u.Email,
		LowerEmail:  strings.ToLower(u.Email),
		IsActivated: u.IsActive,
		IsPrimary:   true,
	}); err != nil {
		return err
	}

	if setting.SbtKeycloakForm.Enabled {
		if err := CreateUserInKeycloak(u, password); err != nil {
			return err
		}
	}

	return committer.Commit()
}
