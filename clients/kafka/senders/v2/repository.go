package v1

import (
	"context"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/modules/log"
	"github.com/IBM/sarama"
	"github.com/google/uuid"

	"code.gitea.io/gitea/clients/kafka"
	"code.gitea.io/gitea/models/events/v2"
	"code.gitea.io/gitea/models/events/v2/generic"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
)

const (
	version = "2.0.0"
)

// CreateRepositorySender - отправляет события о создании репозитория
type CreateRepositorySender struct{}

// CreateRepositorySenderOptions - опции для сообщения о создании репозитория
type CreateRepositorySenderOptions struct {
	repo   *repo.Repository
	tenant *tenant.ScTenant
	doerId string
}

// NewCreateRepositorySender - создает отправителя сообщений о создании репозитория
func NewCreateRepositorySender() CreateRepositorySender {
	return CreateRepositorySender{}
}

// NewCreateRepositorySenderOptions - создает опции для сообщения о создании репозитория
func NewCreateRepositorySenderOptions(repo *repo.Repository, tenant *tenant.ScTenant, doerId string) CreateRepositorySenderOptions {
	return CreateRepositorySenderOptions{
		repo:   repo,
		tenant: tenant,
		doerId: doerId,
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

	metadata := generic.Metadata{
		CorrelationMessageId: nil,
		MessageCreateTs:      strconv.FormatInt(int64(timeutil.TimeStampNow()), 10),
		MessageId:            uuid.NewString(),
		Producer: generic.Producer{
			Id: setting.Kafka.Issuer,
		},
		Version: version,
	}

	contextEvent := generic.Context{
		EntityType:    generic.REPOSITORY,
		EventCode:     generic.CREATE,
		EventCreateTs: strconv.FormatInt(int64(options.repo.CreatedUnix), 10),
		EventId:       uuid.NewString(),
		TenantId:      options.tenant.ID,
	}

	repositoryInfo := events.RepositoryInfo{
		ProjectName:    options.repo.OwnerName,
		RepositoryId:   strconv.FormatInt(options.repo.ID, 10),
		RepositoryName: options.repo.Name,
		TenantName:     options.tenant.Name,
	}

	initiatorUser := events.InitiatorUser{
		Id: options.doerId,
	}

	additionalProperties := make(events.AdditionalProperties)
	additionalProperties["repository_description"] = options.repo.Description
	additionalProperties["repository_uri"] = options.repo.LinkWithoutSub()

	payload := events.Payload{
		RepositoryInfo:       repositoryInfo,
		InitiatorUser:        initiatorUser,
		AdditionalProperties: &additionalProperties,
	}

	createRepositoryEvent := events.CreateRepositoryEvent{
		Context:  contextEvent,
		Metadata: metadata,
		Payload:  payload,
	}

	marshal, err := json.Marshal(createRepositoryEvent)
	if err != nil {
		return fmt.Errorf("error marshaling event: %v", err)
	}

	msg := &sarama.ProducerMessage{
		Value: sarama.ByteEncoder(marshal),
	}

	defer repositoryTopic.Close()

	return repositoryTopic.Produce(msg)
}
