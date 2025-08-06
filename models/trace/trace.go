package trace

import (
	"time"
)

const Key = "TraceID"
const EndpointKey = "TraceEndpoint"
const FrontedKey = "Fronted"

type Message struct {
	TraceId      string                 `json:"traceId"`
	MessageId    string                 `json:"messageId"`
	Method       string                 `json:"method"`
	BeforeMethod string                 `json:"beforeMethod"`
	Fronted      bool                   `json:"fronted"`
	Endpoint     string                 `json:"endpoint"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Start        time.Time              `json:"-"`
}
type TimeMessage struct {
	TraceId   string `json:"traceId"`
	MessageId string `json:"messageId"`
	Method    string `json:"method"`
	Endpoint  string `json:"endpoint"`
	Worktime  string `json:"worktime"`
}

func NewMessage(traceId string, messageId string, frontend bool, endpoint string, method string, beforeMethod string) Message {
	return Message{
		TraceId:      traceId,
		MessageId:    messageId,
		Fronted:      frontend,
		Endpoint:     endpoint,
		Method:       method,
		BeforeMethod: beforeMethod,
		Start:        time.Now(),
	}
}

func NewMessageWithParams(traceId string, messageId string, frontend bool, endpoint string, method string, beforeMethod string, params map[string]interface{}) Message {
	return Message{
		TraceId:      traceId,
		MessageId:    messageId,
		Fronted:      frontend,
		Endpoint:     endpoint,
		Method:       method,
		BeforeMethod: beforeMethod,
		Params:       params,
		Start:        time.Now(),
	}
}

func NewTimeMessage(traceId string, messageId string, method string, endpoint string, startTime time.Time) TimeMessage {
	return TimeMessage{
		TraceId:   traceId,
		MessageId: messageId,
		Method:    method,
		Endpoint:  endpoint,
		Worktime:  time.Now().Sub(startTime).String(),
	}
}

func NewTimeMessageFromTraceMessage(traceMessage Message) TimeMessage {
	return TimeMessage{
		TraceId:   traceMessage.TraceId,
		MessageId: traceMessage.MessageId,
		Method:    traceMessage.Method,
		Endpoint:  traceMessage.Endpoint,
		Worktime:  time.Now().Sub(traceMessage.Start).String(),
	}
}
