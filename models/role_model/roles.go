package role_model

import (
	"sync"

	"code.gitea.io/gitea/modules/json"
)

// Role тип для перечисления ролей
type Role int

// Перечисление ролей
const (
	OWNER Role = iota + 1
	MANAGER
	WRITER
	READER
	TUZ
)

var (
	allRolesMu      sync.RWMutex
	userRolesMu     sync.RWMutex
	userRoleNamesMu sync.RWMutex
)

// Описание ролей
var allRoles = map[Role]string{
	OWNER:   "owner",
	MANAGER: "manager",
	WRITER:  "writer",
	READER:  "reader",
	TUZ:     "tuz",
}

var userRoles = map[Role]string{
	OWNER:   "owner",
	MANAGER: "manager",
	WRITER:  "writer",
	READER:  "reader",
}

var userRoleNames = map[Role]string{
	OWNER:   "Владелец проекта",
	MANAGER: "Менеджер",
	WRITER:  "Пользователь с правами на запись",
	READER:  "Пользователь с правами на чтение",
}

// String возвращает строковое имя роли
func (r Role) String() string {
	allRolesMu.RLock()
	defer allRolesMu.RUnlock()
	return allRoles[r]
}

func (r *Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func GetRoleByString(roleString string) (Role, bool) {
	allRolesMu.RLock()
	defer allRolesMu.RUnlock()
	for role, roleStr := range allRoles {
		if roleStr == roleString {
			return role, true
		}
	}
	return 0, false
}

func GetAllRoles() map[Role]string {
	allRolesMu.RLock()
	defer allRolesMu.RUnlock()
	copy := make(map[Role]string, len(allRoles))
	for k, v := range allRoles {
		copy[k] = v
	}
	return copy
}

func GetUserRoles() map[Role]string {
	userRolesMu.RLock()
	defer userRolesMu.RUnlock()
	copy := make(map[Role]string, len(userRoles))
	for k, v := range userRoles {
		copy[k] = v
	}
	return copy
}

func GetUserRoleNames() map[Role]string {
	userRoleNamesMu.RLock()
	defer userRoleNamesMu.RUnlock()
	copy := make(map[Role]string, len(userRoleNames))
	for k, v := range userRoleNames {
		copy[k] = v
	}
	return copy
}

func DeleteRole(r Role) {
	allRolesMu.Lock()
	defer allRolesMu.Unlock()
	delete(allRoles, r)
}
