package repo

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	repoModule "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/mailer"
	"net/http"
	"time"
)
import "code.gitea.io/gitea/routers/sbt/logger"

// GetCollaboration возвращает список соавторов для репозитория
func GetCollaboration(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	users, err := repoModel.GetCollaborators(ctx, ctx.Repo.Repository.ID, db.ListOptions{})

	if err != nil {
		log.Error("An error occurred while getting collaboration, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	collaborations := make([]response.Collaboration, len(users))

	for i := range users {
		collaborations[i] = convert.ToCollaboration(ctx, users[i])
	}

	ctx.JSON(http.StatusOK, collaborations)
}

// CreateCollaboration создает соавтора с правами на запись
func CreateCollaboration(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	name := ctx.Params(":collaborator")

	if ctx.Repo.Owner.LowerName == name {
		log.Debug("Not able to provide collaborator rights to user: %s, user is already owner of repo: %s", name, ctx.Repo.Repository.Name)
		ctx.JSON(http.StatusBadRequest, apiError.UserIsAlreadyOwner())
		return
	}

	u, err := userModel.GetUserByName(ctx, name)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("User: %s not found", name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(name))
		} else {
			log.Error("An error occurred while getting user by name, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if !u.IsActive {
		log.Debug("Not able to create collaborator, user: %s is not activated", name)
		ctx.JSON(http.StatusBadRequest, apiError.UserNotActivatedError())
		return
	}

	// Организация не может быть соавтором.
	if u.IsOrganization() {
		log.Debug("Not able to create collaborator, user: %s is an organization", name)
		ctx.JSON(http.StatusBadRequest, apiError.UserIsOrganization())
		return
	}

	if got, err := repoModel.IsCollaborator(ctx, ctx.Repo.Repository.ID, u.ID); err == nil && got {
		log.Debug("Not able to create collaborator, user: %s is already collaborator", name)
		ctx.JSON(http.StatusBadRequest, apiError.UserIsAlreadyCollaborator())
		return
	}

	// В случае если репозиторий принадлежит организации
	if ctx.Repo.Repository.Owner.IsOrganization() {
		if isOwner, err := organization.IsOrganizationOwner(ctx, ctx.Repo.Repository.Owner.ID, u.ID); err != nil {
			log.Error("An error occurred while while checking if user is an organization, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		} else if isOwner {
			log.Debug("Not able to provide collaborator rights to user: %s, user is already owner of repo: %s", name, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserIsAlreadyOwner())
			return
		}
	}

	if err = repoModule.AddCollaborator(ctx, ctx.Repo.Repository, u); err != nil {
		log.Error("An error occurred while creating collaboration, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if setting.Service.EnableNotifyMail {
		mailer.SendCollaboratorMail(u, ctx.Doer, ctx.Repo.Repository)
	}

	ctx.JSON(http.StatusCreated, response.Collaboration{
		User: convert.ToUser(ctx, u, nil),
		AccessMode: &response.AccessMode{
			Created: time.Now(),
			Mode:    perm.AccessModeWrite.String(),
			Updated: time.Now(),
		},
	})
}

// ChangeCollaborationAccessMode изменение прав соавтора
func ChangeCollaborationAccessMode(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	name := ctx.Params(":collaborator")
	mode := perm.ParseAccessMode(ctx.Params(":action"))

	u, err := userModel.GetUserByName(ctx, name)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("User: %s not found", name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(name))
		} else {
			log.Error("An error occurred while getting user by name, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if err := repoModel.ChangeCollaborationAccessMode(ctx, ctx.Repo.Repository, u.ID, mode); err != nil {
		log.Error("An error occurred while changing collaboration access mode for user: %s, err: %v", name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}

// DeleteCollaboration удаляет соавтора репозитория
func DeleteCollaboration(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	name := ctx.Params(":collaborator")

	u, err := userModel.GetUserByName(ctx, name)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("User: %s not found", name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(name))
		} else {
			log.Error("An error occurred while getting user by name, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if err := models.DeleteCollaboration(ctx.Repo.Repository, u.ID); err != nil {
		log.Error("An error occurred while deleting collaboration, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
