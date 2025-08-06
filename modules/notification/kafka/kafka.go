package kafka

import (
	"context"
	"fmt"
	"strconv"

	sendersV1 "code.gitea.io/gitea/clients/kafka/senders/v1"
	sendersV2 "code.gitea.io/gitea/clients/kafka/senders/v2"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"

	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/notification/base"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

type kafkaNotifier struct {
	base.NullNotifier
}

var _ base.Notifier = &kafkaNotifier{}

// NewNotifier create a new kafkaNotifier notifier
func NewNotifier() base.Notifier {
	return &kafkaNotifier{}
}

func (a *kafkaNotifier) NotifyCreateRepository(ctx context.Context, doer, u *user_model.User, repo *repo_model.Repository) {
	if setting.Kafka.Enabled {
		createRepositorySenderV1 := sendersV1.NewCreateRepositorySender()
		createRepositorySenderV2 := sendersV2.NewCreateRepositorySender()
		auditParams := map[string]string{
			"repository":    repo.Name,
			"owner":         repo.OwnerName,
			"repository_id": strconv.FormatInt(repo.ID, 10),
		}
		var tenantById *tenant.ScTenant
		if setting.SourceControl.TenantWithRoleModeEnabled {
			var err error
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, repo.OwnerID)
			if err != nil {
				return
			}
			tenantById, err = tenant.GetTenantByID(ctx, tenantId)

			if err := sendCreateRepositoryEvent(ctx, createRepositorySenderV1, createRepositorySenderV2, repo, doer, tenantById); err != nil {
				auditParams["error"] = "Error has occurred while sending event about new repository"
				audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return
			}
			audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
		}
	}
}

func (a *kafkaNotifier) NotifyMigrateRepository(ctx context.Context, doer, u *user_model.User, repo *repo_model.Repository) {
	if setting.Kafka.Enabled {
		createRepositorySenderV1 := sendersV1.NewCreateRepositorySender()
		createRepositorySenderV2 := sendersV2.NewCreateRepositorySender()
		auditParams := map[string]string{
			"repository":    repo.Name,
			"owner":         repo.OwnerName,
			"repository_id": strconv.FormatInt(repo.ID, 10),
		}
		var tenantById *tenant.ScTenant
		if setting.SourceControl.TenantWithRoleModeEnabled {
			var err error
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, repo.OwnerID)
			if err != nil {
				return
			}
			tenantById, err = tenant.GetTenantByID(ctx, tenantId)

			if err := sendCreateRepositoryEvent(ctx, createRepositorySenderV1, createRepositorySenderV2, repo, doer, tenantById); err != nil {
				auditParams["error"] = "Error has occurred while sending event about new repository"
				audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return
			}
			audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
		}
	}
}

func (a *kafkaNotifier) NotifyForkRepository(ctx context.Context, doer *user_model.User, oldRepo, repo *repo_model.Repository) {
	if setting.Kafka.Enabled {
		createRepositorySenderV1 := sendersV1.NewCreateRepositorySender()
		createRepositorySenderV2 := sendersV2.NewCreateRepositorySender()
		auditParams := map[string]string{
			"repository":    repo.Name,
			"owner":         repo.OwnerName,
			"repository_id": strconv.FormatInt(repo.ID, 10),
		}
		var tenantById *tenant.ScTenant
		if setting.SourceControl.TenantWithRoleModeEnabled {
			var err error
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, repo.OwnerID)
			if err != nil {
				return
			}
			tenantById, err = tenant.GetTenantByID(ctx, tenantId)

			if err := sendCreateRepositoryEvent(ctx, createRepositorySenderV1, createRepositorySenderV2, repo, doer, tenantById); err != nil {
				auditParams["error"] = "Error has occurred while sending event about new repository"
				audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return
			}
			audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
		}
	}
}

func (a *kafkaNotifier) NotifyTransferRepository(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, oldOwnerName string) {
	if setting.Kafka.Enabled {
		createRepositorySenderV1 := sendersV1.NewCreateRepositorySender()
		createRepositorySenderV2 := sendersV2.NewCreateRepositorySender()
		auditParams := map[string]string{
			"repository":    repo.Name,
			"owner":         repo.OwnerName,
			"repository_id": strconv.FormatInt(repo.ID, 10),
		}
		var tenantById *tenant.ScTenant
		if setting.SourceControl.TenantWithRoleModeEnabled {
			var err error
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, repo.OwnerID)
			if err != nil {
				return
			}
			tenantById, err = tenant.GetTenantByID(ctx, tenantId)

			if err := sendCreateRepositoryEvent(ctx, createRepositorySenderV1, createRepositorySenderV2, repo, doer, tenantById); err != nil {
				auditParams["error"] = "Error has occurred while sending event about new repository"
				audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return
			}
			audit.CreateAndSendEvent(audit.CreateRepositorySendEvent, doer.Name, strconv.FormatInt(doer.ID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
		}
	}
}

func sendCreateRepositoryEvent(ctx context.Context, createRepositorySenderV1 sendersV1.CreateRepositorySender, createRepositorySenderV2 sendersV2.CreateRepositorySender, repo *repo_model.Repository, doer *user_model.User, tenant *tenant.ScTenant) error {
	// Используем login как ID пользователя во внешней системе
	loginName := doer.LoginName
	if loginName == "" {
		loginName = doer.Name // на случай локальной авторизации
	}

	options := sendersV1.NewCreateRepositorySenderOptions(repo, tenant, loginName)
	if err := createRepositorySenderV1.Send(ctx, options); err != nil {
		return fmt.Errorf("error has occured while sending create repository event with name '%s': %v", repo.Name, err)
	}

	optionsV2 := sendersV2.NewCreateRepositorySenderOptions(repo, tenant, loginName)
	if err := createRepositorySenderV2.Send(ctx, optionsV2); err != nil {
		return fmt.Errorf("error has occured while sending create repository event with name '%s': %v", repo.Name, err)
	}
	return nil
}
