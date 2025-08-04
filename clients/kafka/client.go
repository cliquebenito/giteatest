package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/IBM/sarama"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

var (
	kafkaClient sarama.Client
)

// InitClient инициализирует клиент Kafka
func InitClient(ctx context.Context) error {
	for i := 0; i < setting.Kafka.ConnectRetries; i++ {
		client, err := CreateClient(ctx)
		if err == nil {
			ctx = context.WithValue(ctx, kafkaClientContextKey, client)
			break
		} else if i == setting.Kafka.ConnectRetries-1 {
			return fmt.Errorf("create Kafka client failed: %v", err)
		}
		log.Error("Create Kafka client attempt #%d/%d failed. Error: %v", i+1, setting.Kafka.ConnectRetries, err)
		log.Info("Backing off for %d seconds", int64(setting.Kafka.ConnectBackoff/time.Second))
		time.Sleep(setting.Kafka.ConnectBackoff)
	}
	return nil
}

// GetClient возвращает клиент Kafka
func GetClient(ctx context.Context) sarama.Client {
	return GetEngine(ctx)
}

// CreateClient создает клиент Kafka
func CreateClient(ctx context.Context) (sarama.Client, error) {
	if !setting.Kafka.Enabled {
		return nil, fmt.Errorf("kafka is not enabled")
	}

	client := GetClient(ctx)
	if client == nil {
		var err error
		config, err := NewConfig()
		if err != nil {
			log.Error("Error has occurred while creating Kafka client configuration: %v", err)
			return nil, fmt.Errorf("create Kafka client configuration: %w", err)
		}
		client, err = sarama.NewClient([]string{setting.Kafka.URL}, config)
		if err != nil {
			log.Error("Error has occurred while creating Kafka client connect: %v", err)
			return nil, fmt.Errorf("create Kafka client connect: %w", err)
		}
	}

	SetDefaultClient(ctx, client)
	return client, nil
}

// NewConfig возвращает конфигурацию Kafka
func NewConfig() (*sarama.Config, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	if setting.Kafka.AuthEnabled {
		config.Net.TLS.Enable = true

		cert, err := tls.X509KeyPair([]byte(setting.Kafka.Certificate), []byte(setting.Kafka.PrivateKey))
		if err != nil {
			log.Error("Error has occurred while loading TLS certificate: %v", err)
			return nil, fmt.Errorf("load TLS certificate: %w", err)
		}

		var caCertPool *x509.CertPool = nil
		if setting.Kafka.CARootCertificate != "" {
			caCertPool = x509.NewCertPool()
			caCertPool.AppendCertsFromPEM([]byte(setting.Kafka.CARootCertificate))
		}

		config.Net.TLS.Config = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
	}

	return config, nil
}
