//go:build !correct

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestKafkaSettingsWithoutConfig проверяет настройки кафки без конфигурации секции [kafka]
func TestKafkaSettingsWithoutConfig(t *testing.T) {
	cfg, err := NewConfigProviderFromData(``)
	assert.NoError(t, err)
	loadKafka(cfg)

	assert.Equal(t, false, Kafka.Enabled)
}

// TestKafkaSettingsWithEmptyConfig проверяет настройки кафки с пустой конфигурацией секции [kafka]
func TestKafkaSettingsWithEmptyConfig(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[kafka]
`)
	assert.NoError(t, err)
	loadKafka(cfg)

	assert.Equal(t, false, Kafka.Enabled)
}

// TestKafkaSettingsWithConfig проверяет настройки кафки с конфигурацией секции [kafka] и выключенным топиком репозитория
func TestKafkaSettingsWithConfig(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[kafka]
ENABLED=true
ADDRESS = "00.00.00.000"
PORT = 9092
[kafka.repository]
TOPIC_ENABLED=false
`)
	assert.NoError(t, err)
	loadKafka(cfg)

	topics := make(map[string]TopicConfig)
	topics[KafkaRepositoryTopic] = TopicConfig{
		false,
		"",
		"multiple",
	}

	assert.Equal(t, true, Kafka.Enabled)
	assert.Equal(t, "00.00.00.000", Kafka.Address)
	assert.Equal(t, 9092, Kafka.Port)
	assert.Equal(t, topics, Kafka.Topics)
}

// TestKafkaSettingsWithConfigAndTopic проверяет настройки кафки с конфигурацией секции [kafka] и включенным топиком репозитория
func TestKafkaSettingsWithConfigAndTopic(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[kafka]
ENABLED=true
ADDRESS = "00.00.00.000"
PORT = 9092
[kafka.repository]
TOPIC_ENABLED=true
TOPIC="repository_workflow"
TYPE=produce
`)
	assert.NoError(t, err)
	loadKafka(cfg)

	topics := make(map[string]TopicConfig)
	topics[KafkaRepositoryTopic] = TopicConfig{
		true,
		"repository_workflow",
		"produce",
	}

	assert.Equal(t, true, Kafka.Enabled)
	assert.Equal(t, "00.00.00.000", Kafka.Address)
	assert.Equal(t, 9092, Kafka.Port)
	assert.Equal(t, topics, Kafka.Topics)
}
