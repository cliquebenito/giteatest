package trace

import (
	"context"

	"code.gitea.io/gitea/models/trace"
)

type Tracer interface {
	CreateTraceMessage(ctx context.Context) trace.Message
	CreateTraceMessageWithParams(ctx context.Context, params map[string]interface{}) trace.Message
	Trace(message trace.Message) error
	TraceTime(message trace.Message) error
}
