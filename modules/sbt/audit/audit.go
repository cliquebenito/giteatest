package audit

import (
	"code.gitea.io/gitea/modules/log"
	"github.com/google/uuid"
	"net"
	"os"
	"time"
)

const (
	dateTimeFormatWithMilliseconds = "2006-01-02T15:04:05.000Z07:00" // Формат времени с миллисекундами
	EmptyRequiredField             = "-"                             // Пустое значение для обязательных полей
)

// CreateAndSendEvent функция создающая событие аудита и записывающая событие в файл и консоль
func CreateAndSendEvent(event Event, userName string, userId string, status eventStatus, remoteAddress string, params map[string]string) {
	eventMessage := createEvent(event, userName, userId, status, remoteAddress, params)
	if eventMessage == nil {
		return
	}

	eventMessage.Send(false)
}

// CreateAndSendEventToFile функция создающая событие аудита и записывающая событие только в файл
func CreateAndSendEventToFile(event Event, userName string, userId string, status eventStatus, remoteAddress string, params map[string]string) {
	eventMessage := createEvent(event, userName, userId, status, remoteAddress, params)
	if eventMessage == nil {
		return
	}

	eventMessage.Send(true)
}

// createEvent функция создающая событие аудита
func createEvent(event Event, userName string, userId string, status eventStatus, remoteAddress string, params map[string]string) *message {
	if event == 0 {
		log.Error("Undefined event by user %v", userName)
		return nil
	}

	id, err := uuid.NewUUID()
	if err != nil {
		log.Error("Cannot create id for '%v' audit event, error: %v", event.String(), err)
		return nil
	}

	hostName, err := os.Hostname()
	if err != nil {
		log.Warn("Failed to get hostname for event with id '%v' will use empty value for required field. Error: %v", id, err)
		hostName = EmptyRequiredField
	}

	userIp := EmptyRequiredField
	if remoteAddress != EmptyRequiredField {
		userIp = getUserIp(remoteAddress)
	}

	return &message{
		Id:            id,
		Event:         event,
		Date:          time.Now().UTC().Format(dateTimeFormatWithMilliseconds),
		Username:      userName,
		InternalId:    userId,
		Status:        status,
		UserIp:        userIp,
		HostName:      hostName,
		HostIp:        GetLocalIP(),
		FormatVersion: "1",
		Params:        params,
	}
}

// GetLocalIP функция для получения локального IP
func GetLocalIP() string {
	resultIp := EmptyRequiredField
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		log.Warn("Failed to get interface addresses will use empty value for required field. Error: %v", err)
		return EmptyRequiredField
	}

	for _, addr := range addresses {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				resultIp = ipnet.IP.String()
			}
		}
	}
	return resultIp
}

// getUserIp функция для получения пользовательского IP
func getUserIp(remoteAddress string) string {
	ip, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		if net.ParseIP(remoteAddress) != nil {
			return remoteAddress
		}
		log.Warn("Failed to get user ip for remote address '%v' will use empty value for required field. Error: %v", remoteAddress, err)
		return EmptyRequiredField
	}
	return ip
}
