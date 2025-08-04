package protected_brancher

import (
	"context"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/modules/log"
)

type protectedBranchMerger interface {
	MergeProtectedBranch(context.Context, *protected_branch.ProtectedBranch, *protected_branch.ProtectedBranch) *protected_branch.ProtectedBranch
	MergeProtectedBranchRules(context.Context, protected_branch.ProtectedBranchRules) *protected_branch.ProtectedBranch
}

type ProtectedBranchMerger struct{}

func NewProtectedBranchMerger() *ProtectedBranchMerger {
	return &ProtectedBranchMerger{}
}

// MergeProtectedBranch merges two ProtectedBranch objects into one,
// giving priority to the plain branch configuration.
// Whitelists are combined from both branches.
func (p ProtectedBranchMerger) MergeProtectedBranch(_ context.Context, pb *protected_branch.ProtectedBranch, newpb *protected_branch.ProtectedBranch) *protected_branch.ProtectedBranch {
	return git_model.MergeProtectedBranch(pb, newpb)
}

// Merge all protected branch rules to rules in parameter
func (p ProtectedBranchMerger) MergeProtectedBranchRules(ctx context.Context, rules protected_branch.ProtectedBranchRules) *protected_branch.ProtectedBranch {
	if len(rules) == 0 {
		return nil
	}
	protectedBranch := &protected_branch.ProtectedBranch{}
	log.Debug("Start merge protected branch")
	for _, rule := range rules {
		protectedBranch = p.MergeProtectedBranch(ctx, protectedBranch, rule)
	}
	log.Debug("Protected branch complete")
	return protectedBranch
}
