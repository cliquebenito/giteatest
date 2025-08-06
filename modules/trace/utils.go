package trace

import (
	"context"

	"code.gitea.io/gitea/modules/log"
)

func getStringFromContext(ctx context.Context, key string) string {
	value := ctx.Value(key)
	if value == nil {
		log.Warn("no key %s in context. Setting to empty string", key)
		value = ""
	}
	return value.(string)
}

func getBoolFromContext(ctx context.Context, key string) bool {
	value := ctx.Value(key)
	if value == nil {
		log.Warn("no key %s in context. Setting to false", key)
		value = false
	}
	return value.(bool)
}
