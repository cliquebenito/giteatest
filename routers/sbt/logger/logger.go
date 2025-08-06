package logger

import (
	"code.gitea.io/gitea/modules/context"
	baseLog "code.gitea.io/gitea/modules/log"
	"fmt"
	"os"
)

/*
Для логирования с traceId:
-имплементировать логер "code.gitea.io/gitea/routers/sbt/logger"
-в начале метода с актуальным контекстом создать структуру:
log := log.Logger{}
-установить актуальное значение TraceId
log.SetTraceId(ctx)
В этом файле реализованы функции-обертки для метода /modules/log/BaseLogger.Log()
*/

type Logger struct {
	traceId string
}

// SetTraceId метод, в котором достается TraceId из cookie контекста для дальнейшего логирования
func (l *Logger) SetTraceId(ctx *context.Context) {
	if ctx.Req != nil {
		if traceIdCookie, _ := ctx.Req.Cookie("traceId"); traceIdCookie != nil {
			l.traceId = fmt.Sprintf("[TraceId: %s] ", traceIdCookie.Value)
		}
	}
}

func Log(skip int, level baseLog.Level, format string, traceId string, v ...any) {
	baseLog.GetLogger(baseLog.DEFAULT).Log(skip+1, level, traceId+format, v...)
}

func (l *Logger) Debug(format string, v ...any) {
	Log(1, baseLog.DEBUG, format, l.traceId, v...)
}

func (l *Logger) IsDebug() bool {
	return baseLog.GetLevel() <= baseLog.DEBUG
}

func (l *Logger) Trace(format string, v ...any) {
	Log(1, baseLog.TRACE, format, l.traceId, v...)
}

func (l *Logger) IsTrace() bool {
	return baseLog.GetLevel() <= baseLog.TRACE
}

func (l *Logger) Info(format string, v ...any) {
	Log(1, baseLog.INFO, format, l.traceId, v...)
}

func (l *Logger) Warn(format string, v ...any) {
	Log(1, baseLog.WARN, format, l.traceId, v...)
}

func (l *Logger) Error(format string, v ...any) {
	Log(1, baseLog.ERROR, format, l.traceId, v...)
}

func (l *Logger) Critical(format string, v ...any) {
	Log(1, baseLog.ERROR, format, l.traceId, v...)
}

// Fatal records fatal log and exit process
func (l *Logger) Fatal(format string, v ...any) {
	Log(1, baseLog.FATAL, format, l.traceId, v...)
	baseLog.GetManager().Close()
	os.Exit(1)
}
