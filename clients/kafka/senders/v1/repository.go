package v1

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/google/uuid"

	"code.gitea.io/gitea/clients/kafka"
	"code.gitea.io/gitea/models/events/v1"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

const (
	version = "1.0.0"
)

// CreateRepositorySender - отправляет события о создании репозитория
type CreateRepositorySender struct{}

// CreateRepositorySenderOptions - опции для сообщения о создании репозитория
type CreateRepositorySenderOptions struct {
	repo   *repo.Repository
	tenant *tenant.ScTenant
	userId string
}

// NewCreateRepositorySender - создает отправителя сообщений о создании репозитория
func NewCreateRepositorySender() CreateRepositorySender {
	return CreateRepositorySender{}
}

// NewCreateRepositorySenderOptions - создает опции для сообщения о создании репозитория
func NewCreateRepositorySenderOptions(repo *repo.Repository, tenant *tenant.ScTenant, userId string) CreateRepositorySenderOptions {
	return CreateRepositorySenderOptions{
		repo:   repo,
		tenant: tenant,
		userId: userId,
	}
}

// Send отправляет сообщение о создании репозитория
func (r CreateRepositorySender) Send(ctx context.Context, options CreateRepositorySenderOptions) error {
	log.Debug(`trying to send create repository event with options: %v`, options)
	topicInfo := setting.Kafka.Topics[setting.KafkaRepositoryTopic]
	if !topicInfo.Enabled {
		log.Debug(`repository topic is disabled`)
		return nil
	}

	if topicInfo.Type != string(kafka.Produce) {
		log.Debug(`Produce create repository event is disabled`)
		return nil
	}

	repositoryTopic := kafka.NewTopic(ctx, topicInfo.Enabled, topicInfo.Name, topicInfo.Type)

	projectName := events.Property{
		Name:  "project-name",
		Value: options.repo.Name,
	}

	projectDescription := events.Property{
		Name:  "project-description",
		Value: options.repo.Description,
	}

	uri := events.Property{
		Name:  "uri",
		Value: options.repo.LinkWithoutSub(),
	}

	event := &events.CreateRepositoryEvent{
		Id:     uuid.New().String(),
		Action: events.Create,
		Issuer: setting.Kafka.Issuer,
		ProjectInfo: events.ProjectInfo{
			ProjectId: fmt.Sprintf("/%s/%s/%s", options.tenant.Name, options.repo.OwnerName, options.repo.Name),
		},
		Properties: []events.Property{projectName, projectDescription, uri},
		Timestamp:  int(options.repo.CreatedUnix),
		Type:       events.Node,
		UserInfo: events.UserInfo{
			UserId: options.userId,
		},
		TenantId: options.tenant.ID,
		Version:  version,
	}

	marshal, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling event: %v", err)
	}

	msg := &sarama.ProducerMessage{
		Value: sarama.ByteEncoder(marshal),
	}

	defer repositoryTopic.Close()

	return repositoryTopic.Produce(msg)
}
