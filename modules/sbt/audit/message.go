package audit

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit/writers"
)

// Message cтруктура сообщения события для вывода в консоль и файл в виде json
// https://dzo.sw.sbc.space/wiki/pages/viewpage.action?pageId=250505529
type message struct {
	Id            uuid.UUID         `json:"id"`
	Event         Event             `json:"event"`
	Date          string            `json:"eventdate"`
	Username      string            `json:"username"`
	InternalId    string            `json:"internal_id"`
	Status        eventStatus       `json:"status"`
	UserIp        string            `json:"user_ip"`
	HostName      string            `json:"host.name"`
	HostIp        string            `json:"host.ip"`
	FormatVersion string            `json:"format_version"`
	Params        map[string]string `json:"-"`
}

// MarshalJson функция для преобразования структуры message в "плоский" json
func (e *message) MarshalJson() ([]byte, error) {
	messageFieldMap := make(map[string]string, 0)
	messageBytes, _ := json.Marshal(e)
	err := json.Unmarshal(messageBytes, &messageFieldMap)
	if err != nil {
		log.Error("Cannot unmarshal event to map, error: %v", err)
		return nil, err
	}
	for k, v := range e.Params {
		if v != "" {
			messageFieldMap[k] = v
		}
	}

	return json.Marshal(messageFieldMap)
}

// Send функция вывода сообщения события аудита
func (e message) Send(onlyFile bool) {
	bytes, err := e.MarshalJson()
	if err != nil {
		log.Error("Error marshaling audit params: %s, error: %v", e, err)
		return
	}

	auditionBytesMessage(bytes)
	if !onlyFile {
		fmt.Println(fmt.Sprintf(string(bytes)))
	}
}

// auditionBytesMessage функция для аудирования сообщения в файл
func auditionBytesMessage(message []byte) {
	tmpEvent := &log.Event{
		Time:   time.Now(),
		Caller: "?()",
	}
	formatted := &log.EventFormatted{
		Origin: tmpEvent,
		Msg:    fmt.Sprintln(string(message)),
	}

	auditWriter := writers.NewAuditWriter(log.WriterFileOption{})
	auditWriter.GetAuditWriterQueueForSend() <- formatted
}
