package protected_brancher

import (
	"context"
	"fmt"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/log"
)

// //go:generate mockery --name=managerProtectedBranchDB --exported
type managerProtectedBranchDB interface {
	FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error)
	GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error)
	GetProtectedBranchRuleByID(ctx context.Context, repoID, ruleID int64) (*protected_branch.ProtectedBranch, error)
	UpdateProtectBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error)
	UpsertProtectBranch(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error
	CreateProtectedBranch(ctx context.Context, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error)
	DeleteProtectedBranch(ctx context.Context, repoID, id int64) error
}

type ProtectedBranchManager struct {
	protectedBranchGetter
	protectedBranchChecker
	protectedBranchMerger
	protectedBranchUpdater

	db managerProtectedBranchDB
}

func NewProtectedBranchManager(getter protectedBranchGetter, checker protectedBranchChecker, merger protectedBranchMerger, updater protectedBranchUpdater, db managerProtectedBranchDB) ProtectedBranchManager {
	return ProtectedBranchManager{getter, checker, merger, updater, db}
}

// GetFirstMatched returns the first matching protected branch rule for the given branch name.
func (p ProtectedBranchManager) GetFirstMatched(_ context.Context, rules protected_branch.ProtectedBranchRules, branchName string) *protected_branch.ProtectedBranch {
	return git_model.GetFirstMatched(rules, branchName)
}

// GetMatchProtectedBranchRules returns all matching protected branch rules for a branch.
func (p ProtectedBranchManager) GetMatchProtectedBranchRules(_ context.Context, rules protected_branch.ProtectedBranchRules, branchName string) protected_branch.ProtectedBranchRules {
	return git_model.GetMatchProtectedBranchRules(rules, branchName)
}

// GetMergeMatchProtectedBranchRule returns a merged protected branch rule for the given repo and branch.
func (p ProtectedBranchManager) GetMergeMatchProtectedBranchRule(ctx context.Context, repoID int64, branchName string) (*protected_branch.ProtectedBranch, error) {
	return git_model.GetMergeMatchProtectedBranchRule(ctx, repoID, branchName)
}

// IsBranchProtected checks if branch is protected
func (p ProtectedBranchManager) IsBranchProtected(ctx context.Context, repoID int64, branchName string) (bool, error) {
	return git_model.IsBranchProtected(ctx, repoID, branchName)
}

// FindRepoProtectedBranchRules fetches all protected branch rules for the given repository.
func (p ProtectedBranchManager) FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error) {
	rules, err := p.db.FindRepoProtectedBranchRules(ctx, repoID)
	if err != nil {
		log.Error("Error has occured while find repo protected branch rules: %v", err)
		return nil, fmt.Errorf("Err: find repo protected branch rules: %w", err)
	}

	return rules, nil
}

// GetProtectedBranchRuleByName retrieves a protected branch rule by name.
func (p ProtectedBranchManager) GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error) {
	rule, err := p.db.GetProtectedBranchRuleByName(ctx, repoID, ruleName)
	if err != nil {
		log.Error("Error has occured while get protected branch rule by name: %v", err)
		return nil, fmt.Errorf("Err: get protected branch rule by name: %w", err)
	}
	if rule == nil {
		return nil, NewProtectedBranchNotFoundError()
	}
	return rule, nil
}

// GetProtectedBranchRuleByID retrieves a protected branch rule by its ID.
func (p ProtectedBranchManager) GetProtectedBranchRuleByID(ctx context.Context, repoID, ruleID int64) (*protected_branch.ProtectedBranch, error) {
	rule, err := p.db.GetProtectedBranchRuleByID(ctx, repoID, ruleID)
	if err != nil {
		log.Error("Error has occured while get protected branch rule by id: %v", err)
		return nil, fmt.Errorf("Err: get protected branch rule by id: %w", err)
	}
	if rule == nil {
		return nil, NewProtectedBranchNotFoundError()
	}
	return rule, nil
}

// CreateProtectedBranch creates a new protected branch rule if it doesn't already exist.
func (p ProtectedBranchManager) CreateProtectedBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error) {
	existProtectedBranch, err := p.GetProtectedBranchRuleByName(ctx, repo.ID, protectedBranch.RuleName)
	if err != nil && !IsProtectedBranchNotFoundError(err) {
		log.Error("Error has occured while get protected branch rule by name: %v", err)
		return nil, fmt.Errorf("Err: get protected branch rule by name: %w", err)
	}

	if existProtectedBranch != nil {
		return nil, NewProtectedBranchAlreadyExistError(existProtectedBranch.RuleName)
	}
	p.deleteWhitelistDublicate(protectedBranch)
	protectedBranch.RepoID = repo.ID
	opts := protected_branch.WhitelistOptions{
		UserIDs:          protectedBranch.WhitelistUserIDs,
		MergeUserIDs:     protectedBranch.MergeWhitelistUserIDs,
		ApprovalsUserIDs: protectedBranch.ApprovalsWhitelistUserIDs,
		DeleteUserIDs:    protectedBranch.DeleterWhitelistUserIDs,
		ForcePushUserIDs: protectedBranch.ForcePushWhitelistUserIDs,
	}
	protectedBranch.WhitelistUserIDs = make([]int64, 0)
	protectedBranch.DeleterWhitelistUserIDs = make([]int64, 0)
	protectedBranch.ForcePushWhitelistUserIDs = make([]int64, 0)
	if err = p.UpdateWhitelistOptions(ctx, repo, protectedBranch, opts); err != nil {
		return nil, err
	}

	return p.db.CreateProtectedBranch(ctx, protectedBranch)
}

// UpsertProtectBranch inserts or updates a protected branch rule in the database.
func (p ProtectedBranchManager) UpsertProtectBranch(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error {
	return p.db.UpsertProtectBranch(ctx, repo, protectBranch, opts)
}

// UpdateProtectedBranch updates an existing protected branch rule with new settings.
func (p ProtectedBranchManager) UpdateProtectedBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch, ruleName string) (*protected_branch.ProtectedBranch, error) {
	existProtectedBranch, err := p.GetProtectedBranchRuleByName(ctx, repo.ID, ruleName)
	if err != nil {
		return nil, err
	}

	p.deleteWhitelistDublicate(protectedBranch)
	if err = p.UpdateWhitelistOptions(ctx, repo, existProtectedBranch, protected_branch.WhitelistOptions{
		UserIDs:          protectedBranch.WhitelistUserIDs,
		MergeUserIDs:     protectedBranch.MergeWhitelistUserIDs,
		ApprovalsUserIDs: protectedBranch.ApprovalsWhitelistUserIDs,
		DeleteUserIDs:    protectedBranch.DeleterWhitelistUserIDs,
		ForcePushUserIDs: protectedBranch.ForcePushWhitelistUserIDs,
	}); err != nil {
		return nil, err
	}

	protectedBranch = p.UpdateModelProtectedBranch(existProtectedBranch, protectedBranch)

	return p.db.UpdateProtectBranch(ctx, repo, protectedBranch)
}

// DeleteProtectedBranchByRuleName deletes a protected branch rule by its rule name.
func (p ProtectedBranchManager) DeleteProtectedBranchByRuleName(ctx context.Context, repo *repo_model.Repository, ruleName string) error {
	protectedBranch, err := p.GetProtectedBranchRuleByName(ctx, repo.ID, ruleName)
	if err != nil {
		log.Error("Error has occured while get protected branch rule by name: %v", err)
		return fmt.Errorf("Err: get protected branch rule by name: %w", err)
	}

	return p.db.DeleteProtectedBranch(ctx, repo.ID, protectedBranch.ID)
}

// DeleteProtectedBranch deletes a protected branch rule by its ID.
func (p ProtectedBranchManager) DeleteProtectedBranch(ctx context.Context, repoID, protectedBranchID int64) error {
	return p.db.DeleteProtectedBranch(ctx, repoID, protectedBranchID)
}

// Remove duplicates from the whitelists
func (p ProtectedBranchManager) deleteWhitelistDublicate(protectedBranch *protected_branch.ProtectedBranch) {
	protectedBranch.WhitelistUserIDs = p.deleteDublicate(protectedBranch.WhitelistUserIDs)
	protectedBranch.DeleterWhitelistUserIDs = p.deleteDublicate(protectedBranch.DeleterWhitelistUserIDs)
	protectedBranch.ForcePushWhitelistUserIDs = p.deleteDublicate(protectedBranch.ForcePushWhitelistUserIDs)
}

// Remove duplicates in a slice of int64
func (p ProtectedBranchManager) deleteDublicate(whitelist []int64) []int64 {
	uniqueWhitelist := make(map[int64]struct{})
	for _, id := range whitelist {
		uniqueWhitelist[id] = struct{}{}
	}

	// Convert map keys back to a slice
	result := make([]int64, 0, len(uniqueWhitelist))
	for id := range uniqueWhitelist {
		result = append(result, id)
	}

	return result
}
