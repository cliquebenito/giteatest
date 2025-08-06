package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

	"code.gitea.io/gitea/modules/log"
)

// Topic представляет топик Кафки
type Topic struct {
	// Enabled Активировано ли откидлывание событий в топик
	Enabled bool
	// Name название топика для событий
	Name string
	// Type тип топика
	Type TopicType
	//client Клиент Кафки
	client sarama.Client
	//consumer читатель из кафки
	consumer sarama.Consumer
	//producer писатель в кафку
	producer sarama.SyncProducer
}

// NewTopic создает топик
func NewTopic(ctx context.Context, enabled bool, name string, topicType string) (topic *Topic) {
	topic = &Topic{
		Enabled: enabled,
		Name:    name,
		Type:    TopicType(topicType),
		client:  GetClient(ctx),
	}
	topic.init()
	return topic
}

// init инициализирует топик
func (t *Topic) init() {
	switch t.Type {
	case Multiple:
		t.initConsumer()
		t.initProducer()
	case Consume:
		t.initConsumer()
	case Produce:
		t.initProducer()
	}
	log.Debug("Topic %s initialized", t.Name)
}

// Close закрывает топик
func (t *Topic) Close() {
	switch t.Type {
	case Multiple:
		if err := t.producer.Close(); err != nil {
			log.Error("Failed to close producer: %v", err)
			return
		}
		if err := t.consumer.Close(); err != nil {
			log.Error("Failed to close consumer: %v", err)
			return
		}
	case Consume:
		if err := t.consumer.Close(); err != nil {
			log.Error("Failed to close consumer: %v", err)
			return
		}
	case Produce:
		if err := t.producer.Close(); err != nil {
			log.Error("Failed to close producer: %v", err)
			return
		}
	}
	log.Debug("Topic %s closed", t.Name)
}

// initConsumer инициализирует читателя
func (t *Topic) initConsumer() {

}

// initProducer инициализирует писателя
func (t *Topic) initProducer() {
	var err error
	t.producer, err = sarama.NewSyncProducerFromClient(t.client)
	if err != nil {
		log.Fatal("Failed to create producer: %v", err)
	}
}

// Consume чтение из топика
func (t *Topic) Consume() {

}

// Produce написать в топик
func (t *Topic) Produce(msg *sarama.ProducerMessage) error {
	log.Debug("try to produce message: %#v", msg)
	msg.Topic = t.Name
	partition, offset, err := t.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to produce message: %v", err)
	}

	log.Debug("produce success, partition:", partition, ",offset:", offset)
	return nil
}
