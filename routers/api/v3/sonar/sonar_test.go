//go:build !correct

package sonar

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/sonar/domain"
	"code.gitea.io/gitea/models/sonar/mocks"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/web"
)

func TestCreateSonarSettingsAlreadyExist(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockAPIContext(t, "/api/v3/:tenant/:project/:repo/sonar")
	ctx.SetParams(":tenant", "tenant")
	ctx.SetParams(":project", "project")
	ctx.SetParams(":repo", "repo")

	form := &domain.CreateOrUpdateSonarProjectRequest{
		SonarServerURL:  "https://sonar",
		SonarProjectKey: "proj",
		SonarToken:      "token",
	}
	web.SetForm(ctx, form)

	test.LoadUser(t, ctx, 1)
	ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1, IsAdmin: true})
	mockUC := mocks.NewSonarSettingsUsecaser(t)
	mockUC.
		On("CreateSonarSettings", ctx, *form).
		Return(errors.New("already exist"))
	server := NewSonarServer(mockUC)
	server.CreateSonarSettings(ctx)

	assert.Equal(t, http.StatusConflict, ctx.Resp.Status())
}
func TestGetSonarSettings_OK(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockAPIContext(t, "/api/v3/tenant/project/repo/sonar")
	test.LoadUser(t, ctx, 1)
	test.LoadRepo(t, ctx, 1)
	ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1, IsAdmin: true})

	mockUC := mocks.NewSonarSettingsUsecaser(t)
	mockUC.On("SonarSettings", ctx, ctx.Repo.Repository.ID).
		Return(&repo.ScSonarSettings{ProjectKey: "key"}, nil)

	server := NewSonarServer(mockUC)
	server.SonarSettings(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	mockUC.AssertExpectations(t)
}

func TestUpdateSonarSettings_OK(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockAPIContext(t, "/api/v3/tenant/project/repo/sonar")
	ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1, IsAdmin: true})
	test.LoadRepo(t, ctx, 1)

	form := &domain.CreateOrUpdateSonarProjectRequest{
		SonarServerURL:  "https://sonar",
		SonarProjectKey: "proj",
		SonarToken:      "token",
	}
	web.SetForm(ctx, form)

	mockUC := mocks.NewSonarSettingsUsecaser(t)
	mockUC.On("UpdateSonarSettings", ctx, mock.Anything).
		Return(nil)

	server := NewSonarServer(mockUC)
	server.UpdateSonarSettings(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	mockUC.AssertExpectations(t)
}

func TestDeleteSonarSettings_OK(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockAPIContext(t, "/api/v3/tenant/project/repo/sonar")
	ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1, IsAdmin: true})
	test.LoadRepo(t, ctx, 1)

	mockUC := mocks.NewSonarSettingsUsecaser(t)
	mockUC.On("DeleteSonarSettings", ctx, ctx.Repo.Repository.ID).
		Return(nil)

	server := NewSonarServer(mockUC)
	server.DeleteSonarSettings(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	mockUC.AssertExpectations(t)
}

func TestGetSonarSettings_InternalError(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockAPIContext(t, "/api/v3/tenant/project/repo/sonar")
	test.LoadRepo(t, ctx, 1)
	ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1, IsAdmin: true})

	mockUC := mocks.NewSonarSettingsUsecaser(t)
	mockUC.On("SonarSettings", ctx, ctx.Repo.Repository.ID).
		Return(nil, errors.New("unexpected"))

	server := NewSonarServer(mockUC)
	server.SonarSettings(ctx)
	assert.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
	mockUC.AssertExpectations(t)
}
