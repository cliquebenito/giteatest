package writers

import (
	"context"

	"code.gitea.io/gitea/modules/log"
)

type auditWriterLog struct {
	log.EventWriter
}

var auditWriter *auditWriterLog

// NewAuditWriter функция для создания аудит писателя в файл
func NewAuditWriter(writerOption log.WriterFileOption) *auditWriterLog {
	if auditWriter != nil {
		return auditWriter
	}
	writeMode := log.WriterMode{Level: log.TRACE, WriterOption: writerOption}
	writer, err := log.NewEventWriter("audit", "file", writeMode)
	if err != nil {
		log.Fatal("Error has occurred while creating audit writer with output file %v, error: %v", writerOption.FileName, err)
		return nil
	}

	auditWriter = &auditWriterLog{writer}
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		auditWriter.Run(ctx)
	}()
	return auditWriter
}

// GetAuditWriterQueueForSend функция для получения канала отправляющего аудит события в файл
func (a *auditWriterLog) GetAuditWriterQueueForSend() chan<- *log.EventFormatted {
	return a.Base().Queue
}

// GetAuditWriterName функция для получения имени аудит писателя
func (a *auditWriterLog) GetAuditWriterName() string {
	return a.GetWriterName()
}
