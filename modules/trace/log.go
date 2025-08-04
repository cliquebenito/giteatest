package trace

import (
	"context"
	"fmt"
	"regexp"
	"runtime"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/trace"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
)

const FunctionName = "[a-zA-Z]+$"

var re = regexp.MustCompile(FunctionName)

type logTracer struct{}

func NewLogTracer() logTracer {
	return logTracer{}
}

func (l logTracer) CreateTraceMessage(ctx context.Context) trace.Message {
	// todo DI
	// todo feature flag
	// todo trace notification
	// todo enrich api context
	messageId := uuid.NewString()
	traceId := getStringFromContext(ctx, trace.Key)
	endpoint := getStringFromContext(ctx, trace.EndpointKey)
	fronted := getBoolFromContext(ctx, trace.FrontedKey)

	traceMessage := trace.NewMessage(traceId, messageId, fronted, endpoint, getFunctionName(3), getFunctionName(4))
	return traceMessage
}

func (l logTracer) CreateTraceMessageWithParams(ctx context.Context, params map[string]interface{}) trace.Message {
	messageId := uuid.NewString()
	traceId := getStringFromContext(ctx, trace.Key)
	endpoint := getStringFromContext(ctx, trace.EndpointKey)
	fronted := getBoolFromContext(ctx, trace.FrontedKey)

	traceMessage := trace.NewMessageWithParams(traceId, messageId, fronted, endpoint, getFunctionName(3), getFunctionName(4), params)
	return traceMessage
}

func getFunctionName(level int) string {
	pc := make([]uintptr, 1)
	runtime.Callers(level, pc)
	f := runtime.FuncForPC(pc[0])
	return re.FindString(f.Name())
}

func (l logTracer) Trace(traceMessage trace.Message) error {
	message, err := json.Marshal(traceMessage)
	if err != nil {
		return fmt.Errorf("trace json marshal error %w", err)
	}
	log.Info(string(message))
	return nil
}

func (l logTracer) TraceTime(traceMessage trace.Message) error {
	traceTimeMessage := trace.NewTimeMessageFromTraceMessage(traceMessage)

	message, err := json.Marshal(traceTimeMessage)
	if err != nil {
		return fmt.Errorf("trace json marshal error %w", err)
	}
	log.Info(string(message))
	return nil
}
