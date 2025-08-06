package audit

import "code.gitea.io/gitea/modules/json"

// eventStatus тип для перечисления статусов события
type eventStatus int

// перечисление статусов события
const (
	StatusUnknown eventStatus = iota // Статус события - успешно
	StatusSuccess                    // Статус события - неуспешно
	StatusFailure                    // Статус события - неизвестно
)

// имена статусов
var statusNames = map[eventStatus]string{
	StatusUnknown: "UNKNOWN",
	StatusSuccess: "SUCCESS",
	StatusFailure: "FAIL",
}

// String возвращает имя статуса
func (s eventStatus) String() string {
	return statusNames[s]
}

// MarshalJSON функция для преобразования eventStatus в json
func (s *eventStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}
