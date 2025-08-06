package forms

import (
	"fmt"
)

// count - максимальное количество user key в request
const count = 100

// ApplyPrivilegeRequest структура запроса
type ApplyPrivilegeRequest struct {
	Action ApplyPrivilegeGroups `json:"apply_privilege_groups"`
}

// ApplyPrivilegeGroups структура, объединяющая запрос назначения и удаления привилегий
type ApplyPrivilegeGroups struct {
	Grant  []PrivilegeGroupAssignment `json:"grant"`
	Revoke []PrivilegeGroupAssignment `json:"revoke"`
}

// Validate проверяет корректность запроса
func (a ApplyPrivilegeRequest) Validate() error {
	if len(a.Action.Revoke) == 0 && len(a.Action.Grant) == 0 {
		return fmt.Errorf("request incorrect: no grant or revoke actions specified")
	}
	// Проверка ограничения на количество пользователей
	if len(a.Action.Grant)+len(a.Action.Revoke) > count {
		return fmt.Errorf("request incorrect: maximum %d users allowed per request", count)
	}
	// Проверка ограничения на количество привилегий для каждого пользователя
	for _, grant := range a.Action.Grant {
		if len(grant.PrivilegeGroups) > count {
			return fmt.Errorf("request incorrect: maximum %d privileges allowed per user in grant", count)
		}
	}
	for _, revoke := range a.Action.Revoke {
		if len(revoke.PrivilegeGroups) > count {
			return fmt.Errorf("request incorrect: maximum %d privileges allowed per user in revoke", count)
		}
	}
	return nil
}

// HasOnlyRevokeAndGrant проверяет, что в запросе есть назначение и удаление
func (a ApplyPrivilegeRequest) HasOnlyRevokeAndGrant() bool {
	return len(a.Action.Revoke) > 0 && len(a.Action.Grant) > 0
}

// HasOnlyRevoke проверяет, что в запросе только удаление
func (a ApplyPrivilegeRequest) HasOnlyRevoke() bool {
	return len(a.Action.Revoke) > 0 && len(a.Action.Grant) == 0
}

// HasOnlyGrant проверяет, что в запросе только назначение
func (a ApplyPrivilegeRequest) HasOnlyGrant() bool {
	return len(a.Action.Revoke) == 0 && len(a.Action.Grant) > 0
}

// PrivilegeRequest структура запроса на получение привилегий
type PrivilegeRequest struct {
	TenantKey  string `json:"tenant_key" binding:"Required"`
	ProjectKey string `json:"project_key" binding:"Required"`
	UserKey    string `json:"user_key" binding:"Required"`
}

// GetPrivilegesRequest — массив запросов
type GetPrivilegesRequest []PrivilegeRequest

// Validate проверяет корректность запроса
func (a GetPrivilegesRequest) Validate() error {
	if len(a) == 0 {
		return fmt.Errorf("Err: request incorrect")
	}
	return nil
}

// PrivilegeGroupAssignmentGrant структура назначения привилегий
type PrivilegeGroupAssignmentGrant struct {
	UserExternalID  string                `json:"user_key" binding:"Required"`
	PrivilegeGroups []PrivilegeGroupGrant `json:"privilege_groups"`
}

// PrivilegeGroupGrant структура назначения привилегий
type PrivilegeGroupGrant struct {
	TenantKey      string   `json:"tenant_key" binding:"Required"`
	ProjectKey     string   `json:"project_key" binding:"Required"`
	PrivilegeGroup []string `json:"privilege_group" binding:"Required"` // enum значение
}

// ResponsePrivilegesGet структура ответа для запроса на получение привилегий
type ResponsePrivilegesGet struct {
	Grant []PrivilegeGroupAssignmentGrant `json:"granted"`
}

// ResponsePrivileges структура ответа для запроса на назначение и удаление привилегий
type ResponsePrivileges struct {
	Grant  []PrivilegeGroupAssignment `json:"granted"`
	Revoke []PrivilegeGroupAssignment `json:"revoked"`
}

// PrivilegeGroupAssignment структура назначения привилегий
type PrivilegeGroupAssignment struct {
	UserExternalID  string           `json:"user_key" binding:"Required"`
	PrivilegeGroups []PrivilegeGroup `json:"privilege_groups" binding:"Required"`
}

// PrivilegeGroup структура назначения привилегий
type PrivilegeGroup struct {
	TenantKey       string `json:"tenant_key" binding:"Required"`
	ProjectKey      string `json:"project_key" binding:"Required"`
	PrivilegesGroup string `json:"privilege_group" binding:"Required"`
}

// ApplyPrivilegesResponse структура ответа для запроса на назначение и удаление привилегий с ошибкой
type ApplyPrivilegesResponse struct {
	AppliedStatus ResponsePrivileges      `json:"applied_status"`
	Error         ResponsePrivilegesError `json:"errors"`
}

// Структура для ошибок
type ResponsePrivilegesError struct {
	Grant  []PrivilegeGroupAssignmentErr `json:"grant,omitempty"`
	Revoke []PrivilegeGroupAssignmentErr `json:"revoke,omitempty"`
}

// Структура ошибки для пользователя
type PrivilegeGroupAssignmentErr struct {
	UserExternalID  string              `json:"user_key"`
	PrivilegeGroups []PrivilegeGroupErr `json:"privilege_groups"`
}

// Структура ошибки для группы привилегий
type PrivilegeGroupErr struct {
	TenantID       string `json:"tenant_key"`
	ProjectKey     string `json:"project_key"`
	PrivilegeGroup string `json:"privilege_group"`
	ErrMsg         string `json:"error"`
}
