//go:build !correct

package tests

import (
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/routers/sbt/user"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

/*
Для запуска тестов необходимо добавить в настройках тестов "Go tool arguments": -tags sqlite,sqlite_unlock_notify
Для запуска тестов из консоли использовать команду go test code.gitea.io/gitea/routers/sbt/user/tests -tags sqlite,sqlite_unlock_notify
Тестовые пользователи загружаются в контекст функцией test.LoadUser.
Данные загружаемых пользователей можно посмотреть в models/fixtures/user.yml
*/

/*
TestUpdateUserSettings_changeAllParam тест проверяет изменились ли данные настроек пользователя после метода
UpdateUserSettings, вернул ли метод статус 200(ок), изменился ли контекст пользователя и сам пользователь
*/
func TestUpdateUserSettings_changeAllParam(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockContext(t, "/user/settings")
	test.LoadUser(t, ctx, 3)

	username := "NewUser6"
	fullName := "new test user four"
	website := "qwerty"
	location := "Moscow"
	description := "gitVerse's test user"
	visibility := "limited"
	keepEmailPrivate := true
	keepActivityPrivate := true

	opts := request.UserSettingsOptional{
		Name:                &username,
		FullName:            &fullName,
		Website:             &website,
		Location:            &location,
		Description:         &description,
		Visibility:          &visibility,
		KeepEmailPrivate:    &keepEmailPrivate,
		KeepActivityPrivate: &keepActivityPrivate,
	}

	assert.NotEqualValues(t, username, ctx.Doer.Name)
	assert.NotEqualValues(t, fullName, ctx.Doer.FullName)
	assert.NotEqualValues(t, website, ctx.Doer.Website)
	assert.NotEqualValues(t, location, ctx.Doer.Location)
	assert.NotEqualValues(t, description, ctx.Doer.Description)
	assert.NotEqualValues(t, visibility, ctx.Doer.Visibility.String())
	assert.NotEqualValues(t, keepEmailPrivate, ctx.Doer.KeepEmailPrivate)
	assert.NotEqualValues(t, keepActivityPrivate, ctx.Doer.KeepActivityPrivate)

	web.SetForm(ctx, &opts)
	user.UpdateUserSettings(ctx)

	assert.EqualValues(t, username, ctx.Doer.Name)
	assert.NotEqualValues(t, username, ctx.Doer.LowerName)
	assert.EqualValues(t, strings.ToLower(username), ctx.Doer.LowerName)
	assert.EqualValues(t, fullName, ctx.Doer.FullName)
	assert.EqualValues(t, website, ctx.Doer.Website)
	assert.EqualValues(t, location, ctx.Doer.Location)
	assert.EqualValues(t, description, ctx.Doer.Description)
	assert.EqualValues(t, visibility, ctx.Doer.Visibility.String())
	assert.EqualValues(t, keepEmailPrivate, ctx.Doer.KeepEmailPrivate)
	assert.EqualValues(t, keepActivityPrivate, ctx.Doer.KeepActivityPrivate)

	assert.EqualValues(t, http.StatusOK, ctx.Resp.Status())

	ctxAfter := test.MockContext(t, "/user/settings")
	test.LoadUser(t, ctxAfter, 3)

	assert.EqualValues(t, username, ctxAfter.Doer.Name)
	assert.EqualValues(t, fullName, ctxAfter.Doer.FullName)
	assert.EqualValues(t, website, ctxAfter.Doer.Website)
	assert.EqualValues(t, location, ctxAfter.Doer.Location)
	assert.EqualValues(t, description, ctxAfter.Doer.Description)
	assert.EqualValues(t, visibility, ctxAfter.Doer.Visibility.String())
	assert.EqualValues(t, keepEmailPrivate, ctxAfter.Doer.KeepEmailPrivate)
	assert.EqualValues(t, keepActivityPrivate, ctxAfter.Doer.KeepActivityPrivate)
}

/*
TestUpdateUserSettings_changeOneParam тест проверяет изменятся ли другие поля настроек пользователя
при изменении одного параметра.
*/
func TestUpdateUserSettings_changeOneParam(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockContext(t, "/user/settings")
	test.LoadUser(t, ctx, 2)

	var fullName string

	opts := request.UserSettingsOptional{
		FullName: &fullName,
	}

	assert.NotEqualValues(t, fullName, ctx.Doer.FullName)

	ctxBeforeUpdate := test.MockContext(t, "/user/settings")
	test.LoadUser(t, ctxBeforeUpdate, 2)

	web.SetForm(ctx, &opts)

	user.UpdateUserSettings(ctx)

	assert.NotEqualValues(t, ctxBeforeUpdate.Doer.FullName, ctx.Doer.FullName)
	assert.EqualValues(t, fullName, ctx.Doer.FullName)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.Name, ctx.Doer.Name)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.Website, ctx.Doer.Website)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.Location, ctx.Doer.Location)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.Description, ctx.Doer.Description)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.Visibility, ctx.Doer.Visibility)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.KeepEmailPrivate, ctx.Doer.KeepEmailPrivate)
	assert.EqualValues(t, ctxBeforeUpdate.Doer.KeepActivityPrivate, ctx.Doer.KeepActivityPrivate)
	assert.EqualValues(t, http.StatusOK, ctx.Resp.Status())
}

/*
TestUpdateUserSettings_changeUserNameTable тест проверяет несколько кейсов изменения имени пользователя
*/
func TestUpdateUserSettings_changeUserNameTable(t *testing.T) {
	for _, userNameTable := range []struct {
		username         string
		expectedRespCode int
	}{
		{
			username:         "newTestUserName",
			expectedRespCode: 200,
		},
		{
			username:         "user",
			expectedRespCode: 400,
		},
		{
			username:         "user2",
			expectedRespCode: 400,
		},
		{
			username:         "*",
			expectedRespCode: 400,
		},
		{
			username:         "",
			expectedRespCode: 400,
		},
	} {
		t.Run(userNameTable.username, func(t *testing.T) {
			unittest.PrepareTestEnv(t)

			ctx := test.MockContext(t, "/user/settings")
			test.LoadUser(t, ctx, 1)

			opts := request.UserSettingsOptional{
				Name: &userNameTable.username,
			}
			assert.NotEqualValues(t, userNameTable.username, ctx.Doer.Name)

			doerNameBefore := ctx.Doer.Name
			web.SetForm(ctx, &opts)
			user.UpdateUserSettings(ctx)

			assert.EqualValues(t, userNameTable.expectedRespCode, ctx.Resp.Status())
			if userNameTable.expectedRespCode == 200 {
				assert.EqualValues(t, userNameTable.username, ctx.Doer.Name)
				assert.NotEqualValues(t, doerNameBefore, ctx.Doer.Name)
				assert.NotEqualValues(t, userNameTable.username, ctx.Doer.LowerName)
				assert.EqualValues(t, strings.ToLower(userNameTable.username), ctx.Doer.LowerName)
			} else {
				assert.NotEqualValues(t, userNameTable.username, ctx.Doer.Name)
				assert.NotEqualValues(t, strings.ToLower(userNameTable.username), ctx.Doer.LowerName)
				assert.EqualValues(t, doerNameBefore, ctx.Doer.Name)
			}
		})
	}
}
