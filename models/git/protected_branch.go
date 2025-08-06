// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/models/git/protected_branch/utils"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"

	"github.com/gobwas/glob"
)

// IsRuleNameSpecial return true if it contains special character
// Deprecated: use ProtectedBranchManager.isRuleNameSpecial
func IsRuleNameSpecial(ruleName string) bool {
	return utils.IsRuleNameSpecial(ruleName)
}

// Deprecated: use ProtectedBranchManager.GetGlob
func LoadGlob(protectBranch protected_branch.ProtectedBranch) (glob.Glob, bool) {
	return protectBranch.LoadGlob()
}

// CanUserPush returns if some user could push to this protected branch
// Deprecated: use ProtectedBranchManager.CheckUserCanPush
func CanUserPush(_ context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool {
	if protectBranch.EnableWhitelist {
		return base.Int64sContains(protectBranch.WhitelistUserIDs, user.ID)
	}

	return true
}

// IsUserMergeWhitelisted checks if some user is whitelisted to merge to this branch
// Deprecated: use ProtectedBranchManager.IsUserMergeWhitelisted
func IsUserMergeWhitelisted(ctx context.Context, protectBranch protected_branch.ProtectedBranch, userID int64, permissionInRepo access_model.Permission) bool {
	if !protectBranch.EnableMergeWhitelist {
		// Then we need to fall back on whether the user has write permission
		return permissionInRepo.CanWrite(unit.TypeCode)
	}

	return base.Int64sContains(protectBranch.MergeWhitelistUserIDs, userID)
}

// IsUserOfficialReviewer check if user is official reviewer for the branch (counts towards required approvals)
// Deprecated: use ProtectedBranchManager.IsUserOfficialReviewer
func IsUserOfficialReviewer(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool {
	if protectBranch.EnableApprovalsWhitelist {
		return base.Int64sContains(protectBranch.ApprovalsWhitelistUserIDs, user.ID)
	}

	return true
}

// GetProtectedFilePatterns parses a semicolon separated list of protected file patterns and returns a glob.Glob slice
// Deprecated: use ProtectedBranchManager.GetProtectedFilePatterns
func GetProtectedFilePatterns(protectBranch protected_branch.ProtectedBranch) []glob.Glob {
	return getFilePatterns(protectBranch.ProtectedFilePatterns)
}

// GetUnprotectedFilePatterns parses a semicolon separated list of unprotected file patterns and returns a glob.Glob slice
// Deprecated: use ProtectedBranchManager.GetUnprotectedFilePatterns
func GetUnprotectedFilePatterns(protectBranch protected_branch.ProtectedBranch) []glob.Glob {
	return getFilePatterns(protectBranch.UnprotectedFilePatterns)
}

func getFilePatterns(filePatterns string) []glob.Glob {
	extarr := make([]glob.Glob, 0, 10)
	for _, expr := range strings.Split(strings.ToLower(filePatterns), ";") {
		expr = strings.TrimSpace(expr)
		if expr != "" {
			if g, err := glob.Compile(expr, '.', '/'); err != nil {
				log.Info("Invalid glob expression '%s' (skipped): %v", expr, err)
			} else {
				extarr = append(extarr, g)
			}
		}
	}
	return extarr
}

// MergeBlockedByProtectedFiles returns true if merge is blocked by protected files change
// Deprecated: use ProtectedBranchManager.MergeBlockedByProtectedFiles
func MergeBlockedByProtectedFiles(protectBranch protected_branch.ProtectedBranch, changedProtectedFiles []string) bool {
	glob := GetProtectedFilePatterns(protectBranch)
	if len(glob) == 0 {
		return false
	}

	return len(changedProtectedFiles) > 0
}

// IsProtectedFile return if path is protected
// Deprecated: use ProtectedBranchManager.IsProtectedFile
func IsProtectedFile(protectBranch protected_branch.ProtectedBranch, patterns []glob.Glob, path string) bool {
	if len(patterns) == 0 {
		patterns = GetProtectedFilePatterns(protectBranch)
		if len(patterns) == 0 {
			return false
		}
	}

	lpath := strings.ToLower(strings.TrimSpace(path))

	r := false
	for _, pat := range patterns {
		if pat.Match(lpath) {
			r = true
			break
		}
	}

	return r
}

// IsUnprotectedFile return if path is unprotected
// Deprecated: use ProtectedBranchManager.MergeBlockedByProtectedFiles
func IsUnprotectedFile(protectBranch protected_branch.ProtectedBranch, patterns []glob.Glob, path string) bool {
	if len(patterns) == 0 {
		patterns = GetUnprotectedFilePatterns(protectBranch)
		if len(patterns) == 0 {
			return false
		}
	}

	lpath := strings.ToLower(strings.TrimSpace(path))

	r := false
	for _, pat := range patterns {
		if pat.Match(lpath) {
			r = true
			break
		}
	}

	return r
}

// MergeProtectedBranch merges two ProtectedBranch objects into one,
// giving priority to the plain branch configuration.
// Whitelists are combined from both branches.
// Deprecated: use ProtectedBranchManager.MergeProtectedBranch
func MergeProtectedBranch(pb *protected_branch.ProtectedBranch, newpb *protected_branch.ProtectedBranch) *protected_branch.ProtectedBranch {
	if pb == nil {
		log.Debug("MergeProtectedBranch: base branch is nil")
		return nil
	}
	if newpb == nil {
		log.Debug("MergeProtectedBranch: new branch is nil")
		return pb
	}

	ID := newpb.ID
	repoID := newpb.RepoID
	repo := newpb.Repo
	ruleName := newpb.RuleName
	globRule := newpb.GlobRule
	isPlainName := newpb.IsPlainName
	createdUnix := newpb.CreatedUnix
	updatedUnix := newpb.UpdatedUnix
	if pb.IsPlainName {
		ID = pb.ID
		repoID = pb.RepoID
		repo = pb.Repo
		ruleName = pb.RuleName
		globRule = pb.GlobRule
		isPlainName = pb.IsPlainName
		createdUnix = pb.CreatedUnix
		updatedUnix = pb.UpdatedUnix
	}

	protectedFilePatterns := joinPatterns(pb.ProtectedFilePatterns, newpb.ProtectedFilePatterns)
	unprotectedFilePatterns := joinPatterns(pb.UnprotectedFilePatterns, newpb.UnprotectedFilePatterns)

	// settings for push
	enableWhitelist := pb.EnableWhitelist || newpb.EnableWhitelist
	whiteListUserIDs := mergeWhiteLists(enableWhitelist, pb.WhitelistUserIDs, newpb.WhitelistUserIDs)
	whitelistDeployKeys := pb.WhitelistDeployKeys || newpb.WhitelistDeployKeys
	requireSignedCommits := pb.RequireSignedCommits || newpb.RequireSignedCommits

	// setting for force push
	enableForcePushWhitelist := pb.EnableForcePushWhitelist || newpb.EnableForcePushWhitelist
	forcePushWhitelistUserIDs := mergeWhiteLists(enableForcePushWhitelist, pb.ForcePushWhitelistUserIDs, newpb.ForcePushWhitelistUserIDs)
	forcePushWhitelstDeployKeys := pb.ForcePushWhitelistDeployKeys || newpb.ForcePushWhitelistDeployKeys

	// settings for branch deletions
	enableDeleterWhitelist := pb.EnableDeleterWhitelist || newpb.EnableDeleterWhitelist
	deleterWhitelistUserIDs := mergeWhiteLists(enableDeleterWhitelist, pb.DeleterWhitelistUserIDs, newpb.DeleterWhitelistUserIDs)
	deleterWhitelistDeployKeys := pb.DeleterWhitelistDeployKeys || newpb.DeleterWhitelistDeployKeys

	// settings for approvals
	requiredApprovals := pb.RequiredApprovals + newpb.RequiredApprovals
	enableApprovalsWhiteList := pb.EnableApprovalsWhitelist || newpb.EnableApprovalsWhitelist
	approvalWhitelistUserIDs := mergeWhiteLists(enableApprovalsWhiteList, pb.ApprovalsWhitelistUserIDs, newpb.ApprovalsWhitelistUserIDs)
	dismissStaleApprovals := pb.DismissStaleApprovals || newpb.DismissStaleApprovals
	enableStatusCheck := pb.EnableStatusCheck || newpb.EnableStatusCheck
	statusCheckContexts := mergeStringLists(enableStatusCheck, pb.StatusCheckContexts, newpb.StatusCheckContexts)

	// settings for merge
	enableMergeWhitelist := pb.EnableMergeWhitelist || newpb.EnableMergeWhitelist
	mergeWhitelistUserIDs := mergeWhiteLists(enableMergeWhitelist, pb.MergeWhitelistUserIDs, newpb.MergeWhitelistUserIDs)
	blockOnRejectedviews := pb.BlockOnRejectedReviews || newpb.BlockOnRejectedReviews
	blockOnOfficialReviewRequests := pb.BlockOnOfficialReviewRequests || newpb.BlockOnOfficialReviewRequests
	blockOnOutdatedBranch := pb.BlockOnOutdatedBranch || newpb.BlockOnOutdatedBranch
	enableSonarQube := pb.EnableSonarQube || newpb.EnableSonarQube

	return &protected_branch.ProtectedBranch{
		ID:          ID,
		RepoID:      repoID,
		Repo:        repo,
		RuleName:    ruleName,
		GlobRule:    globRule,
		IsPlainName: isPlainName,

		ProtectedFilePatterns:   protectedFilePatterns,
		UnprotectedFilePatterns: unprotectedFilePatterns,

		EnableWhitelist:      enableWhitelist,
		WhitelistUserIDs:     whiteListUserIDs,
		WhitelistDeployKeys:  whitelistDeployKeys,
		RequireSignedCommits: requireSignedCommits,

		EnableForcePushWhitelist:     enableForcePushWhitelist,
		ForcePushWhitelistUserIDs:    forcePushWhitelistUserIDs,
		ForcePushWhitelistDeployKeys: forcePushWhitelstDeployKeys,

		EnableDeleterWhitelist:     enableDeleterWhitelist,
		DeleterWhitelistUserIDs:    deleterWhitelistUserIDs,
		DeleterWhitelistDeployKeys: deleterWhitelistDeployKeys,

		RequiredApprovals:         requiredApprovals,
		EnableApprovalsWhitelist:  enableApprovalsWhiteList,
		ApprovalsWhitelistUserIDs: approvalWhitelistUserIDs,
		DismissStaleApprovals:     dismissStaleApprovals,
		EnableStatusCheck:         enableStatusCheck,
		StatusCheckContexts:       statusCheckContexts,

		EnableMergeWhitelist:          enableMergeWhitelist,
		MergeWhitelistUserIDs:         mergeWhitelistUserIDs,
		BlockOnRejectedReviews:        blockOnRejectedviews,
		BlockOnOfficialReviewRequests: blockOnOfficialReviewRequests,
		BlockOnOutdatedBranch:         blockOnOutdatedBranch,
		EnableSonarQube:               enableSonarQube,

		CreatedUnix: createdUnix,
		UpdatedUnix: updatedUnix,
	}
}

func joinPatterns(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + ";" + b
	}
}

func mergeStringLists(flag bool, oldWhiteList, newWhiteList []string) []string {
	result := make([]string, 0)
	if !flag {
		return result
	}

	uniqueElements := make(map[string]struct{})

	for _, id := range oldWhiteList {
		uniqueElements[id] = struct{}{}
	}
	for _, id := range newWhiteList {
		uniqueElements[id] = struct{}{}
	}

	for id := range uniqueElements {
		result = append(result, id)
	}

	return result
}

func mergeWhiteLists(flag bool, oldWhiteList, newWhiteList []int64) []int64 {
	result := make([]int64, 0)
	if !flag {
		return result
	}

	uniqueElements := make(map[int64]struct{})

	for _, id := range oldWhiteList {
		uniqueElements[id] = struct{}{}
	}
	for _, id := range newWhiteList {
		uniqueElements[id] = struct{}{}
	}

	for id := range uniqueElements {
		result = append(result, id)
	}

	return result
}

// GetProtectedBranchRuleByName getting protected branch rule by name
// Deprecated: use protectedBranchDB.GetProtectedBranchRuleByName
func GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error) {
	rel := &protected_branch.ProtectedBranch{RepoID: repoID, RuleName: ruleName}
	has, err := db.GetByBean(ctx, rel)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return rel, nil
}

// GetProtectedBranchRuleByID getting protected branch rule by rule ID
// Deprecated: use protectedBranchDB.GetProtectedBranchRuleByID
func GetProtectedBranchRuleByID(ctx context.Context, repoID, ruleID int64) (*protected_branch.ProtectedBranch, error) {
	rel := &protected_branch.ProtectedBranch{ID: ruleID, RepoID: repoID}
	has, err := db.GetByBean(ctx, rel)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return rel, nil
}

// UpdateProtectBranch saves branch protection options of repository.
// If ID is 0, it creates a new record. Otherwise, updates existing record.
// This function also performs check if whitelist user and team's IDs have been changed
// to avoid unnecessary whitelist delete and regenerate.
// Deprecated: use protectedBranchDB.UpdateProtectBranch
func UpdateProtectBranch(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) (err error) {
	if err = repo.LoadOwner(ctx); err != nil {
		return fmt.Errorf("LoadOwner: %v", err)
	}

	if err = UpdateWhitelistOptions(ctx, repo, protectBranch, opts); err != nil {
		return fmt.Errorf("Error update white lists: %v", err)
	}

	// Make sure protectBranch.ID is not 0 for whitelists
	if protectBranch.ID == 0 {
		if _, err = db.GetEngine(ctx).Insert(protectBranch); err != nil {
			return fmt.Errorf("Insert: %v", err)
		}
		return nil
	}

	if _, err = db.GetEngine(ctx).ID(protectBranch.ID).AllCols().Update(protectBranch); err != nil {
		return fmt.Errorf("Update: %v", err)
	}

	return nil
}

// Deprecated: use protecteBrancUpdater.UpdateWhitelistOptions
func UpdateWhitelistOptions(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error {
	whitelist, err := updateUserWhitelist(ctx, repo, protectBranch.WhitelistUserIDs, opts.UserIDs)
	if err != nil {
		return err
	}
	protectBranch.WhitelistUserIDs = whitelist

	whitelist, err = updateUserWhitelist(ctx, repo, protectBranch.MergeWhitelistUserIDs, opts.MergeUserIDs)
	if err != nil {
		return err
	}
	protectBranch.MergeWhitelistUserIDs = whitelist

	whitelist, err = updateApprovalWhitelist(ctx, repo, protectBranch.ApprovalsWhitelistUserIDs, opts.ApprovalsUserIDs)
	if err != nil {
		return err
	}
	protectBranch.ApprovalsWhitelistUserIDs = whitelist

	whitelist, err = updateUserWhitelist(ctx, repo, protectBranch.DeleterWhitelistUserIDs, opts.DeleteUserIDs)
	if err != nil {
		return err
	}
	protectBranch.DeleterWhitelistUserIDs = whitelist

	whitelist, err = updateUserWhitelist(ctx, repo, protectBranch.ForcePushWhitelistUserIDs, opts.ForcePushUserIDs)
	if err != nil {
		return err
	}
	protectBranch.ForcePushWhitelistUserIDs = whitelist

	return nil
}

// updateApprovalWhitelist checks whether the user whitelist changed and returns a whitelist with
// the users from newWhitelist which have explicit read or write access to the repo.
func updateApprovalWhitelist(ctx context.Context, repo *repo_model.Repository, currentWhitelist, newWhitelist []int64) (whitelist []int64, err error) {
	hasUsersChanged := !util.SliceSortedEqual(currentWhitelist, newWhitelist)
	if !hasUsersChanged {
		return currentWhitelist, nil
	}

	whitelist = make([]int64, 0, len(newWhitelist))
	for _, userID := range newWhitelist {
		if reader, err := access_model.IsRepoReader(ctx, repo, userID); err != nil {
			return nil, err
		} else if !reader {
			continue
		}
		whitelist = append(whitelist, userID)
	}

	return whitelist, err
}

// updateUserWhitelist checks whether the user whitelist changed and returns a whitelist with
// the users from newWhitelist which have write access to the repo.
func updateUserWhitelist(ctx context.Context, repo *repo_model.Repository, currentWhitelist, newWhitelist []int64) (whitelist []int64, err error) {
	hasUsersChanged := !util.SliceSortedEqual(currentWhitelist, newWhitelist)
	if !hasUsersChanged {
		return currentWhitelist, nil
	}

	whitelist = make([]int64, 0, len(newWhitelist))
	for _, userID := range newWhitelist {
		user, err := user_model.GetUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("GetUserByID [user_id: %d, repo_id: %d]: %v", userID, repo.ID, err)
		}
		perm, err := access_model.GetUserRepoPermission(ctx, repo, user)
		if err != nil {
			return nil, fmt.Errorf("GetUserRepoPermission [user_id: %d, repo_id: %d]: %v", userID, repo.ID, err)
		}

		if !perm.CanWrite(unit.TypeCode) {
			continue // Drop invalid user ID
		}

		whitelist = append(whitelist, userID)
	}

	return whitelist, err
}

// DeleteProtectedBranch removes ProtectedBranch relation between the user and repository.
// Deprecated: use protectedBranchDB.DeleteProtectedBranch
func DeleteProtectedBranch(ctx context.Context, repoID, id int64) (err error) {
	protectedBranch := &protected_branch.ProtectedBranch{
		RepoID: repoID,
		ID:     id,
	}

	if affected, err := db.GetEngine(ctx).Delete(protectedBranch); err != nil {
		return err
	} else if affected != 1 {
		return fmt.Errorf("delete protected branch ID(%v) failed", id)
	}

	return nil
}

// RemoveUserIDFromProtectedBranch remove all user ids from protected branch options
// Deprecated: use protectedBranchDB.RemoveUserIDFromProtectedBranch
func RemoveUserIDFromProtectedBranch(ctx context.Context, p *protected_branch.ProtectedBranch, userID int64) error {
	lenIDs,
		lenApprovalIDs,
		lenMergeIDs,
		lenForcePushIDs,
		lenDeleteIDs :=
		len(p.WhitelistUserIDs),
		len(p.ApprovalsWhitelistUserIDs),
		len(p.MergeWhitelistUserIDs),
		len(p.ForcePushWhitelistUserIDs),
		len(p.DeleterWhitelistUserIDs)

	p.WhitelistUserIDs = util.SliceRemoveAll(p.WhitelistUserIDs, userID)
	p.ApprovalsWhitelistUserIDs = util.SliceRemoveAll(p.ApprovalsWhitelistUserIDs, userID)
	p.MergeWhitelistUserIDs = util.SliceRemoveAll(p.MergeWhitelistUserIDs, userID)
	p.ForcePushWhitelistUserIDs = util.SliceRemoveAll(p.ForcePushWhitelistUserIDs, userID)
	p.DeleterWhitelistUserIDs = util.SliceRemoveAll(p.DeleterWhitelistUserIDs, userID)

	if lenIDs != len(p.WhitelistUserIDs) ||
		lenApprovalIDs != len(p.ApprovalsWhitelistUserIDs) ||
		lenMergeIDs != len(p.MergeWhitelistUserIDs) ||
		lenForcePushIDs != len(p.ForcePushWhitelistUserIDs) ||
		lenDeleteIDs != len(p.DeleterWhitelistUserIDs) {
		if _, err := db.GetEngine(ctx).ID(p.ID).Cols(
			"whitelist_user_i_ds",
			"merge_whitelist_user_i_ds",
			"approvals_whitelist_user_i_ds",
			"deleter_whitelist_user_i_ds",
			"force_push_whitelist_user_i_ds",
		).Update(p); err != nil {
			return fmt.Errorf("updateProtectedBranches: %v", err)
		}
	}
	return nil
}
