package role_model

import (
	"code.gitea.io/gitea/modules/json"
)

// Action тип для перечисления действий
type Action int

// Перечисление действий
const (
	OWN Action = iota + 1
	CREATE
	EDIT
	EDIT_PROJECT
	READ
	READ_PRIVATE
	WRITE
	DELETE
	MERGE_WITHOUT_CHECK
	MANAGE_COMMENTS
)

// Описание действий
var actions = map[Action]string{
	OWN:                 "own",
	CREATE:              "create",
	EDIT:                "edit",
	EDIT_PROJECT:        "edit_project",
	READ:                "read",
	READ_PRIVATE:        "read_private",
	WRITE:               "write",
	DELETE:              "delete",
	MERGE_WITHOUT_CHECK: "merge_without_check",
	MANAGE_COMMENTS:     "manage_comments",
}

// String возвращает описание действий
func (a Action) String() string {
	return actions[a]
}

// MarshalJSON функция для преобразования Action в json
func (a *Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// GetActionByString метод для получения Action по строке
func GetActionByString(str string) (Action, bool) {
	for action, actionString := range actions {
		if actionString == str {
			return action, true
		}
	}
	return 0, false
}
