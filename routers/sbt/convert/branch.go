package convert

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	gitModel "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/perm"
	accessModel "code.gitea.io/gitea/models/perm/access"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/sbt/response"
)

// ToBranch конвертирует git.Commit и git.Branch в *response.Branch
func ToBranch(ctx context.Context, repo *repoModel.Repository, b *git.Branch, c *git.Commit, bp *protected_branch.ProtectedBranch, user *userModel.User, isRepoAdmin bool) (*response.Branch, error) {
	if bp == nil {
		var hasPerm bool
		var canPush bool
		var err error
		if user != nil {
			hasPerm, err = accessModel.HasAccessUnit(db.DefaultContext, user, repo, unit.TypeCode, perm.AccessModeWrite)
			if err != nil {
				log.Error("Error has occured while check perm: %v", err)
				return nil, fmt.Errorf("Err: has access unit: %w", err)
			}

			perms, err := accessModel.GetUserRepoPermission(db.DefaultContext, repo, user)
			if err != nil {
				log.Error("Error has occured while get user repo permission: %v", err)
				return nil, fmt.Errorf("Err: get user repo permission: %w", err)
			}
			canPush = issuesModel.CanMaintainerWriteToBranch(perms, b.Name, user)
		}

		return &response.Branch{
			Name:                b.Name,
			Commit:              ToPayloadCommit(ctx, c),
			Protected:           false,
			RequiredApprovals:   0,
			EnableStatusCheck:   false,
			StatusCheckContexts: []string{},
			UserCanPush:         canPush,
			UserCanMerge:        hasPerm,
		}, nil
	}

	branch := &response.Branch{
		Name:                b.Name,
		Commit:              ToPayloadCommit(ctx, c),
		Protected:           true,
		RequiredApprovals:   bp.RequiredApprovals,
		EnableStatusCheck:   bp.EnableStatusCheck,
		StatusCheckContexts: bp.StatusCheckContexts,
	}

	if isRepoAdmin {
		branch.EffectiveBranchProtectionName = bp.RuleName
	}

	if user != nil {
		permission, err := accessModel.GetUserRepoPermission(db.DefaultContext, repo, user)
		if err != nil {
			log.Error("Error has occured while get user repo permission: %v", err)
			return nil, fmt.Errorf("Err: get user repo permission: %w", err)
		}
		bp.Repo = repo
		branch.UserCanPush = gitModel.CanUserPush(db.DefaultContext, *bp, user)
		branch.UserCanMerge = gitModel.IsUserMergeWhitelisted(db.DefaultContext, *bp, user.ID, permission)
	}

	return branch, nil
}
