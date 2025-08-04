package create_default

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/user"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/structs"
	"context"
	"fmt"
	gouuid "github.com/google/uuid"
)

// CreateProject - создание проектов из репозиторием и присвоение ролей соавторам и владельцам репозиториев.
func CreateProject(ctx context.Context) error {

	var allRepos []*repo.Repository
	// находим все репозитоории привязанные к пользователям.
	err := db.GetEngine(ctx).
		Join("LEFT", "`user`", "`repository`.owner_id = `user`.id").
		Where("`user`.is_active").
		And("`user`.type = ?", user.UserTypeIndividual).
		Find(&allRepos)

	if err != nil {
		log.Error("There are %d repos already created. ", len(allRepos))
		return err
	}

	// если нет репозиториев с пользователями -- выходим.
	if len(allRepos) == 0 {
		log.Debug("All repo is recreate project.")
		return nil
	}

	defaultTenant, err := CreateTenant(ctx)
	if err != nil {
		return err
	}

	for _, foundRepo := range allRepos {

		err = foundRepo.LoadOwner(ctx)
		if err != nil {
			log.Error("Error found in loading owner. ", err)
			return err
		}

		var repoPrivateStatus = structs.VisibleTypePublic
		if foundRepo.IsPrivate {
			repoPrivateStatus = structs.VisibleTypePrivate
		}

		if !foundRepo.Owner.CanCreateOrganization() {
			log.Debug(fmt.Sprintf("Permission for create organization succes granted for user with id=%d and name=%s", foundRepo.OwnerID, foundRepo.OwnerName))
			foundRepo.Owner.AllowCreateOrganization = true
		}

		projectName := foundRepo.Name
		isExist, err := user_model.IsUserExist(db.DefaultContext, 0, foundRepo.Name)
		if err != nil {
			log.Error("Error has occurred while trying check user exist. Error: %v", err)
			return err
		} else if isExist {
			uuid := gouuid.New()
			projectName += "." + uuid.String()
			log.Debug(fmt.Sprintf("Project name '%s' is exist, using new project name '%s'", foundRepo.Name, projectName))
		}

		org := &organization.Organization{
			Name:       projectName,
			Visibility: repoPrivateStatus,
			IsActive:   true,
		}

		err = organization.CreateOrganization(org, foundRepo.Owner)

		if err != nil {
			log.Error("Error found in creating organization. ", err)
			return err
		}

		metaInfoRepo := repo.Repository{
			ID:        foundRepo.ID,
			Name:      foundRepo.Name,
			OwnerID:   org.ID,
			OwnerName: org.Name,
			Owner:     foundRepo.Owner,
		}

		err = models.TransferOwnership(foundRepo.Owner, org.Name, &metaInfoRepo, false)

		if err != nil {
			return err
		}

		err = FindSequenses(ctx)
		if err != nil {
			return err
		}

		allCollaboration, err := repo.GetCollaborators(ctx, org.ID, db.ListOptions{})
		if err != nil {
			return err
		}

		for _, collaborator := range allCollaboration {
			var targetRule role_model.Role

			switch collaborator.Collaboration.Mode {
			case perm.AccessModeWrite:
				targetRule = role_model.WRITER
			case perm.AccessModeRead:
				targetRule = role_model.READER
			case perm.AccessModeAdmin:
				targetRule = role_model.MANAGER
			case perm.AccessModeOwner:
				targetRule = role_model.OWNER
			default:
				targetRule = role_model.READER
			}

			err = role_model.GrantUserPermissionToOrganization(
				collaborator.User,
				defaultTenant.ID,
				org,
				targetRule,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
