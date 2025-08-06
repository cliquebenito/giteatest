// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package routing

import (
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
)

// NewLoggerHandler is a handler that will log routing to the router log taking account of
// routing information
func NewLoggerHandler() func(next http.Handler) http.Handler {
	manager := requestRecordsManager{
		requestRecords: map[uint64]*requestRecord{},
	}
	manager.startSlowQueryDetector(3 * time.Second)

	return manager.handler
}

var (
	startMessage          = log.NewColoredValue("started  ", log.DEBUG.ColorAttributes()...)
	slowMessage           = log.NewColoredValue("slow     ", log.WARN.ColorAttributes()...)
	pollingMessage        = log.NewColoredValue("polling  ", log.INFO.ColorAttributes()...)
	failedMessage         = log.NewColoredValue("failed   ", log.WARN.ColorAttributes()...)
	completedMessage      = log.NewColoredValue("completed", log.INFO.ColorAttributes()...)
	unknownHandlerMessage = log.NewColoredValue("completed", log.ERROR.ColorAttributes()...)
)

func logPrinter(logger log.Logger, body []byte) func(trigger Event, record *requestRecord) {
	return func(trigger Event, record *requestRecord) {
		// when a request starts, we have no information about the handler function information, we only have the request path
		req := &http.Request{}
		if record != nil && record.request != nil {
			req = record.request
		}

		if trigger == StartEvent {
			switch logger.GetLevel() {
			case log.DEBUG:
				logger.Debug("router: %s %v %s for %s headers: [%+v]", startMessage, log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr, req.Header)
			case log.TRACE:
				logger.Trace("router: %s %v %s for %s headers: [%+v] request body: [%s]", startMessage, log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr, req.Header, body)
			default:
				logger.Info("router: %s %v %s for %s", startMessage, log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr)
			}

			return
		}

		// Get data from the record
		record.lock.Lock()
		handlerFuncInfo := record.funcInfo.String()
		isLongPolling := record.isLongPolling
		isUnknownHandler := record.funcInfo == nil
		panicErr := record.panicError
		record.lock.Unlock()

		if trigger == StillExecutingEvent {
			message := slowMessage
			logf := logger.Warn
			if isLongPolling {
				logf = logger.Info
				message = pollingMessage
			}

			switch logger.GetLevel() {
			case log.DEBUG:
				logger.Debug("router: %s %v %s for %s, elapsed %v @ %s headers: [%+v]",
					message,
					log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
					log.ColoredTime(time.Since(record.startTime)),
					handlerFuncInfo,
					req.Header,
				)
			case log.TRACE:
				logger.Trace("router: %s %v %s for %s, elapsed %v @ %s headers: [%+v] request body: [%s]",
					message,
					log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
					log.ColoredTime(time.Since(record.startTime)),
					handlerFuncInfo,
					req.Header,
					body,
				)
			default:
				logf("router: %s %v %s for %s, elapsed %v @ %s",
					message,
					log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
					log.ColoredTime(time.Since(record.startTime)),
					handlerFuncInfo,
				)
			}

			return
		}

		if panicErr != nil {
			switch logger.GetLevel() {
			case log.DEBUG:
				logger.Debug("router: %s %v %s for %s, panic in %v @ %s headers: [%+v], err=%v",
					failedMessage,
					log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
					log.ColoredTime(time.Since(record.startTime)),
					handlerFuncInfo,
					req.Header,
					panicErr,
				)
			case log.TRACE:
				logger.Trace("router: %s %v %s for %s, panic in %v @ %s headers: [%+v] request body: [%s], err=%v",
					failedMessage,
					log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
					log.ColoredTime(time.Since(record.startTime)),
					handlerFuncInfo,
					req.Header,
					body,
					panicErr,
				)
			default:
				logger.Warn("router: %s %v %s for %s, panic in %v @ %s, err=%v",
					failedMessage,
					log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
					log.ColoredTime(time.Since(record.startTime)),
					handlerFuncInfo,
					panicErr,
				)
			}

			return
		}

		var status int
		if v, ok := record.responseWriter.(context.ResponseWriter); ok {
			status = v.Status()
		}
		logf := logger.Info
		if strings.HasPrefix(req.RequestURI, "/assets/") {
			logf = logger.Trace
		}
		message := completedMessage
		if isUnknownHandler {
			logf = logger.Error
			message = unknownHandlerMessage
		}

		switch logger.GetLevel() {
		case log.DEBUG:
			logger.Debug("router: %s %v %s for %s, %v %v in %v @ %s headers: [%+v]",
				message,
				log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
				log.ColoredStatus(status), log.ColoredStatus(status, http.StatusText(status)), log.ColoredTime(time.Since(record.startTime)),
				handlerFuncInfo,
				req.Header,
			)
		case log.TRACE:
			logger.Trace("router: %s %v %s for %s, %v %v in %v @ %s headers: [%+v] request body: [%s]",
				message,
				log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
				log.ColoredStatus(status), log.ColoredStatus(status, http.StatusText(status)), log.ColoredTime(time.Since(record.startTime)),
				handlerFuncInfo,
				req.Header,
				body,
			)
		default:
			logf("router: %s %v %s for %s, %v %v in %v @ %s",
				message,
				log.ColoredMethod(req.Method), req.RequestURI, req.RemoteAddr,
				log.ColoredStatus(status), log.ColoredStatus(status, http.StatusText(status)), log.ColoredTime(time.Since(record.startTime)),
				handlerFuncInfo,
			)
		}
	}
}
