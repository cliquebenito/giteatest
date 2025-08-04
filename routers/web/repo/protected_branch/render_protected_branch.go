package protected_branch

import (
	"net/http"
	"strings"
	"time"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	access_model "code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"
)

// ProtectedBranchRules render the page to protect the repository
func (s Server) ProtectedBranchRules(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.branches")
	ctx.Data["PageIsSettingsBranches"] = true

	rules, err := s.protectedBranchManager.FindRepoProtectedBranchRules(ctx, ctx.Repo.Repository.ID)
	if err != nil && !protected_brancher.IsProtectedBranchNotFoundError(err) {
		ctx.ServerError("GetProtectedBranches", err)
		return
	}
	ctx.Data["ProtectedBranches"] = rules

	ctx.HTML(http.StatusOK, tplBranches)
}

// SettingsProtectedBranch renders the protected branch setting page
func (s Server) SettingsProtectedBranch(c *context.Context) {
	ruleName := c.FormString("rule_name")
	var rule *protected_branch.ProtectedBranch
	if ruleName != "" {
		var err error
		rule, err = s.protectedBranchManager.GetProtectedBranchRuleByName(c, c.Repo.Repository.ID, ruleName)
		if err != nil && !protected_brancher.IsProtectedBranchNotFoundError(err) {
			c.ServerError("GetProtectBranchOfRepoByName", err)
			return
		}
	}

	if rule == nil {
		// No options found, create defaults.
		rule = &protected_branch.ProtectedBranch{}
	}

	c.Data["PageIsSettingsBranches"] = true
	c.Data["Title"] = c.Tr("repo.settings.protected_branch") + " - " + rule.RuleName

	users, err := access_model.GetRepoReaders(c.Repo.Repository)
	if err != nil {
		c.ServerError("Repo.Repository.GetReaders", err)
		return
	}
	c.Data["Users"] = users
	c.Data["whitelist_users"] = strings.Join(base.Int64sToStrings(rule.WhitelistUserIDs), ",")
	c.Data["merge_whitelist_users"] = strings.Join(base.Int64sToStrings(rule.MergeWhitelistUserIDs), ",")
	c.Data["approvals_whitelist_users"] = strings.Join(base.Int64sToStrings(rule.ApprovalsWhitelistUserIDs), ",")
	c.Data["deleter_whitelist_users"] = strings.Join(base.Int64sToStrings(rule.DeleterWhitelistUserIDs), ",")
	c.Data["force_pusher_whitelist_users"] = strings.Join(base.Int64sToStrings(rule.ForcePushWhitelistUserIDs), ",")
	c.Data["status_check_contexts"] = strings.Join(rule.StatusCheckContexts, "\n")
	contexts, _ := git_model.FindRepoRecentCommitStatusContexts(c, c.Repo.Repository.ID, 7*24*time.Hour) // Find last week status check contexts
	c.Data["recent_status_checks"] = contexts

	c.Data["Rule"] = rule
	c.HTML(http.StatusOK, tplProtectedBranch)
}
