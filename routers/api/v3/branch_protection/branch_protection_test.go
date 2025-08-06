package protected_branch

import (
	"errors"
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v3/branch_protection/mocks"
	"code.gitea.io/gitea/routers/api/v3/models"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"

	"github.com/stretchr/testify/require"
)

var mockManager *mocks.ProtectedBranchManager
var mockConverter *mocks.BranchProtectionConverter
var mockAuditConverter *mocks.AuditConverter
var server Server

func init() {
	mockManager = new(mocks.ProtectedBranchManager)
	mockConverter = new(mocks.BranchProtectionConverter)
	mockAuditConverter = new(mocks.AuditConverter)
	server = NewBranchProtectionServer(mockManager, mockConverter, mockAuditConverter)
}

func setContext(ctx *context.APIContext) *context.APIContext {
	ctx.Repo = &context.Repository{}
	ctx.Repo.Repository = &repo_model.Repository{ID: 1}
	ctx.Doer = &user.User{ID: 1, IsAdmin: true}
	scTenantOrganiztion := &tenant.ScTenantOrganizations{ID: "1", TenantID: "1", OrganizationID: 1, OrgKey: "1", ProjectKey: "1"}
	ctx.Tenant = scTenantOrganiztion
	return ctx
}

func TestGetBranchProtections_Success(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	protectionRules := protected_branch.ProtectedBranchRules{
		{
			ID:               1,
			RepoID:           repo.ID,
			RuleName:         "main",
			WhitelistUserIDs: []int64{1, 2},
		},
	}

	mockManager.On("FindRepoProtectedBranchRules", ctx, repo.ID).Return(protectionRules, nil)
	mockConverter.On("ToBranchProtectionRulesBody", protectionRules).Return([]models.BranchProtectionBody{})

	server.GetBranchProtections(ctx)

	require.Equal(t, http.StatusOK, ctx.Resp.Status())
}

func TestGetBranchProtections_RuleNotFound(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	ctx = setContext(ctx)

	repository := ctx.Repo.Repository

	mockManager.On("FindRepoProtectedBranchRules", ctx, repository.ID).Return(nil, protected_brancher.NewProtectedBranchNotFoundError())

	server.GetBranchProtections(ctx)

	require.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}

func TestGetBranchProtections_InternalError(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	mockManager.On("FindRepoProtectedBranchRules", ctx, repo.ID).Return(nil, errors.New("database connection error"))

	server.GetBranchProtections(ctx)

	require.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}

func TestGetBranchProtection_Success(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	ctx.SetParams(":branch_name", "fakeName")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	branchName := "fakeName"
	fakeRule := &protected_branch.ProtectedBranch{RuleName: branchName}

	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(fakeRule, nil)
	mockConverter.On("ToBranchProtectionBody", *fakeRule).Return(models.BranchProtectionBody{})
	server.GetBranchProtection(ctx)

	require.Equal(t, http.StatusOK, ctx.Resp.Status())
}

func TestGetBranchProtection_RuleNotFound(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	ctx.SetParams(":branch_name", "fakeName")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	branchName := "fakeName"

	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(nil, protected_brancher.NewProtectedBranchNotFoundError())

	server.GetBranchProtection(ctx)

	require.Equal(t, http.StatusNotFound, ctx.Resp.Status())
}

func TestGetBranchProtection_InternalError(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	ctx.SetParams(":branch_name", "fakeName")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	branchName := "fakeName"

	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(nil, errors.New("unexpected internal error"))

	server.GetBranchProtection(ctx)

	require.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}

func TestDeleteBranchProtection_Success(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	ctx.SetParams(":branch_name", "main")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	branchName := "main"
	fakeRule := &protected_branch.ProtectedBranch{RuleName: branchName}
	auditProtectBranch := protected_branch.AuditProtectedBranch{}

	mockAuditConverter.On("Convert", *fakeRule).Return(auditProtectBranch)
	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(fakeRule, nil)
	mockManager.On("DeleteProtectedBranchByRuleName", ctx, repo, branchName).Return(nil)

	server.DeleteBranchProtection(ctx)

	require.Equal(t, http.StatusNoContent, ctx.Resp.Status())
}

func TestDeleteBranchProtection_BranchNotFound(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	branchName := "nonexistent-branch"
	ctx.SetParams(":branch_name", branchName)
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	fakeRule := &protected_branch.ProtectedBranch{RuleName: branchName}
	auditProtectBranch := protected_branch.AuditProtectedBranch{}

	mockAuditConverter.On("Convert", *fakeRule).Return(auditProtectBranch)
	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(nil, protected_brancher.NewProtectedBranchNotFoundError())
	server.DeleteBranchProtection(ctx)

	require.Equal(t, http.StatusNotFound, ctx.Resp.Status())
}

func TestDeleteBranchProtection_InternalError(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	ctx.SetParams(":branch_name", "main")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	branchName := "main"

	fakeRule := &protected_branch.ProtectedBranch{RuleName: branchName}
	auditProtectBranch := protected_branch.AuditProtectedBranch{}

	mockAuditConverter.On("Convert", *fakeRule).Return(auditProtectBranch)
	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(fakeRule, nil)
	mockManager.On("DeleteProtectedBranchByRuleName", ctx, repo, branchName).Return(errors.New("internal error"))

	server.DeleteBranchProtection(ctx)

	require.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}

func TestCreateBranchProtection_Success(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository
	branchName := "main"

	body := models.BranchProtectionBody{
		BranchName: branchName,
	}
	web.SetForm(ctx, &body)

	convertedPB := &protected_branch.ProtectedBranch{RuleName: branchName}

	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockAuditConverter.On("Convert", *convertedPB).Return(auditProtectBranch)
	mockConverter.On("ToProtectedBranch", ctx, body).Return(convertedPB)
	mockManager.On("CreateProtectedBranch", ctx, repo, convertedPB).Return(convertedPB, nil)
	mockConverter.On("ToBranchProtectionBody", *convertedPB).Return(models.BranchProtectionBody{})

	server.CreateBranchProtection(ctx)

	require.Equal(t, http.StatusCreated, ctx.Resp.Status())
}

func TestCreateBranchProtection_ValidationFailedOne(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	branchName := "main"
	ctx = setContext(ctx)

	body := models.BranchProtectionBody{
		BranchName: branchName,
		PushSettings: models.PushSettings{
			RequirePushWhitelist:   false,
			PushWhitelistUsernames: []string{},
		},
	}
	convertedPB := &protected_branch.ProtectedBranch{
		RuleName:         branchName,
		WhitelistUserIDs: []int64{},
		EnableWhitelist:  false,
	}

	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockAuditConverter.On("Convert", *convertedPB).Return(auditProtectBranch)
	mockConverter.On("ToProtectedBranch", ctx, body).Return(convertedPB)

	web.SetForm(ctx, &body)
	server.CreateBranchProtection(ctx)

	require.Equal(t, http.StatusBadRequest, ctx.Resp.Status())
}

func TestCreateBranchProtection_ValidationFailedTwo(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	branchName := "main"
	ctx = setContext(ctx)

	body := models.BranchProtectionBody{
		BranchName: branchName,
		PushSettings: models.PushSettings{
			RequirePushWhitelist: true,
		},
	}

	convertedPB := &protected_branch.ProtectedBranch{
		RuleName:        branchName,
		EnableWhitelist: true,
	}

	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockAuditConverter.On("Convert", *convertedPB).Return(auditProtectBranch)
	mockConverter.On("ToProtectedBranch", ctx, body).Return(convertedPB)
	web.SetForm(ctx, &body)
	server.CreateBranchProtection(ctx)

	require.Equal(t, http.StatusBadRequest, ctx.Resp.Status())
}

func TestCreateBranchProtection_Conflict(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	branchName := "main"
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository
	body := models.BranchProtectionBody{
		BranchName: branchName,
	}

	web.SetForm(ctx, &body)

	convertedPB := &protected_branch.ProtectedBranch{
		RuleName: branchName,
	}

	mockConverter.On("ToProtectedBranch", ctx, body).Return(convertedPB)

	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockAuditConverter.On("Convert", *convertedPB).Return(auditProtectBranch)
	conflictErr := protected_brancher.NewProtectedBranchAlreadyExistError(branchName)
	mockManager.On("CreateProtectedBranch", ctx, repo, convertedPB).Return(nil, conflictErr)

	server.CreateBranchProtection(ctx)

	require.Equal(t, http.StatusConflict, ctx.Resp.Status())
}

func TestCreateBranchProtection_InternalError(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections")
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	body := models.BranchProtectionBody{
		BranchName: "main",
	}

	web.SetForm(ctx, &body)

	convertedPB := &protected_branch.ProtectedBranch{
		RuleName: "main",
	}

	mockConverter.On("ToProtectedBranch", ctx, body).Return(convertedPB)

	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockAuditConverter.On("Convert", *convertedPB).Return(auditProtectBranch)
	internErr := errors.New("internal storage failure")
	mockManager.On("CreateProtectedBranch", ctx, repo, convertedPB).Return(nil, internErr)

	server.CreateBranchProtection(ctx)

	require.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}

func TestUpdateBranchProtection_Success(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	branchName := "main"
	ctx.SetParams(":branch_name", branchName)
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	updateData := models.BranchProtectionBody{
		BranchName: branchName,
		PushSettings: models.PushSettings{
			RequirePushWhitelist:   true,
			PushWhitelistUsernames: []string{"alice", "bob"},
		},
	}

	web.SetForm(ctx, &updateData)

	updatedPB := &protected_branch.ProtectedBranch{
		RuleName:         branchName,
		EnableWhitelist:  true,
		WhitelistUserIDs: []int64{1, 2},
	}

	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(updatedPB, nil)
	mockAuditConverter.On("Convert", *updatedPB).Return(auditProtectBranch)
	mockConverter.On("ToProtectedBranch", ctx, updateData).Return(updatedPB)
	mockManager.On("UpdateProtectedBranch", ctx, repo, updatedPB, branchName).Return(updatedPB, nil)
	mockConverter.On("ToBranchProtectionBody", *updatedPB).Return(models.BranchProtectionBody{})

	server.UpdateBranchProtection(ctx)

	require.Equal(t, http.StatusOK, ctx.Resp.Status())
}

func TestUpdateBranchProtection_NotFound(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	branchName := "nonexistent-branch"
	ctx.SetParams(":branch_name", branchName)
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	updateData := models.BranchProtectionBody{
		BranchName: branchName,
	}

	web.SetForm(ctx, &updateData)

	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(nil, protected_brancher.NewProtectedBranchNotFoundError())

	server.UpdateBranchProtection(ctx)

	require.Equal(t, http.StatusNotFound, ctx.Resp.Status())
}

func TestUpdateBranchProtection_InternalError(t *testing.T) {
	ctx := test.MockAPIContext(t, "/api/v3/repos/tenant/project/repo/branch_protections/:branch_name")
	branchName := "main"
	ctx.SetParams(":branch_name", branchName)
	ctx = setContext(ctx)
	repo := ctx.Repo.Repository

	updateData := models.BranchProtectionBody{
		BranchName: branchName,
	}

	web.SetForm(ctx, &updateData)

	pb := &protected_branch.ProtectedBranch{
		RuleName: branchName,
	}
	auditProtectBranch := protected_branch.AuditProtectedBranch{}
	mockManager.On("GetProtectedBranchRuleByName", ctx, repo.ID, branchName).Return(pb, nil)
	mockAuditConverter.On("Convert", *pb).Return(auditProtectBranch)
	mockConverter.On("ToProtectedBranch", ctx, updateData).Return(pb)
	internErr := errors.New("internal database error")
	mockManager.On("UpdateProtectedBranch", ctx, repo, pb, branchName).Return(nil, internErr)

	server.UpdateBranchProtection(ctx)

	require.Equal(t, http.StatusInternalServerError, ctx.Resp.Status())
}
