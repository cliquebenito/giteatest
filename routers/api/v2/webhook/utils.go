package webhook

import (
	"errors"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/api/v2/models"
)

type ErrIdIsRequired struct {
}

// Реализация интерфейса error
func (err ErrIdIsRequired) Error() string {
	return fmt.Sprintf("Err: id is required")
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrIdIsRequired) Unwrap() error {
	return errors.New("errors id is required")
}

// IsErrIDRequired проверяет, является ли ошибка ErrIdIsRequired
func IsErrIDRequired(err error) bool {
	return errors.As(err, &ErrIdIsRequired{})
}

// getHookID - получение id хука из query params
func getHookID(ctx *context.APIContext) (int64, error) {
	id := ctx.FormString("id")
	if id == "" {
		return 0, ErrIdIsRequired{}
	}
	hookID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, models.ErrInvalidHookID{HookID: id}
	}
	return hookID, nil
}
