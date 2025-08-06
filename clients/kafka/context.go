package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

// DefaultContext — это контекст по умолчанию для запуска запросов xorm
// будет перезаписан Init с HammerContext
var DefaultContext context.Context

// contextKey — это значение для использования с context.WithValue.
type contextKey struct {
	name string
}

// kafkaClientContextKey — контекстный ключ. Он используется с context.Value() для получения текущего Engineed для контекста.
var (
	kafkaClientContextKey        = &contextKey{"kafkaClientContextKey"}
	_                     Client = &Context{}
)

// Context представляет контекст БД
type Context struct {
	context.Context
	contextKafkaClient sarama.Client
}

// Client возвращает базу данных Client
func (ctx *Context) Client() sarama.Client {
	return ctx.contextKafkaClient
}

// Value Значение для context.Context, но позволяет нам получить себя и объект Engineed.
func (ctx *Context) Value(key interface{}) interface{} {
	if key == kafkaClientContextKey {
		return ctx
	}
	return ctx.Context.Value(key)
}

// Client структуры предоставляют KafkaClient
type Client interface {
	Client() sarama.Client
}

// GetEngine получит Клиента БД из этого контекста или вернет Клиента, ограниченного этим контекстом
func GetEngine(ctx context.Context) sarama.Client {
	if e := getEngine(ctx); e != nil {
		return e
	}
	return kafkaClient
}

// getEngine получит клиент БД из этого контекста или вернет ноль
func getEngine(ctx context.Context) sarama.Client {
	if client, ok := ctx.(Client); ok {
		return client.Client()
	}
	clientInterface := ctx.Value(kafkaClientContextKey)
	if clientInterface != nil {
		return clientInterface.(Client).Client()
	}
	return nil
}

// SetDefaultEngine устанавливает движок по умолчанию для БД
func SetDefaultClient(ctx context.Context, client sarama.Client) {
	kafkaClient = client
	DefaultContext = &Context{
		Context:            ctx,
		contextKafkaClient: kafkaClient,
	}
}
