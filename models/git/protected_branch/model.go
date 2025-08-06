package protected_branch

import (
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/git/protected_branch/utils"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/gobwas/glob"
)

func init() {
	db.RegisterModel(new(ProtectedBranch))
}

// ProtectedBranch структура для полей таблицы защиты ветки
type ProtectedBranch struct {
	ID                            int64                  `xorm:"pk autoincr"`
	RepoID                        int64                  `xorm:"UNIQUE(s)"`
	Repo                          *repo_model.Repository `xorm:"-"`
	RuleName                      string                 `xorm:"'branch_name' UNIQUE(s)"` // a branch name or a glob match to branch name
	GlobRule                      glob.Glob              `xorm:"-"`
	IsPlainName                   bool                   `xorm:"-"`
	EnableWhitelist               bool
	WhitelistUserIDs              []int64  `xorm:"JSON TEXT"`
	EnableMergeWhitelist          bool     `xorm:"NOT NULL DEFAULT false"`
	WhitelistDeployKeys           bool     `xorm:"NOT NULL DEFAULT false"`
	MergeWhitelistUserIDs         []int64  `xorm:"JSON TEXT"`
	EnableStatusCheck             bool     `xorm:"NOT NULL DEFAULT false"`
	StatusCheckContexts           []string `xorm:"JSON TEXT"`
	EnableApprovalsWhitelist      bool     `xorm:"NOT NULL DEFAULT false"`
	ApprovalsWhitelistUserIDs     []int64  `xorm:"JSON TEXT"`
	RequiredApprovals             int64    `xorm:"NOT NULL DEFAULT 0"`     // delete in default reviewers
	BlockOnRejectedReviews        bool     `xorm:"NOT NULL DEFAULT false"` // delete in default reviewers
	BlockOnOfficialReviewRequests bool     `xorm:"NOT NULL DEFAULT false"` // delete in default reviewers
	BlockOnOutdatedBranch         bool     `xorm:"NOT NULL DEFAULT false"` // delete in default reviewers
	DismissStaleApprovals         bool     `xorm:"NOT NULL DEFAULT false"` // delete in default reviewers
	RequireSignedCommits          bool     `xorm:"NOT NULL DEFAULT false"`
	ProtectedFilePatterns         string   `xorm:"TEXT"`
	UnprotectedFilePatterns       string   `xorm:"TEXT"`
	EnableSonarQube               bool     `xorm:"NOT NULL DEFAULT false"` // delete in default reviewers

	EnableDeleterWhitelist     bool    `xorm:"NOT NULL DEFAULT false"`
	DeleterWhitelistUserIDs    []int64 `xorm:"JSON TEXT"`
	DeleterWhitelistDeployKeys bool    `xorm:"NOT NULL DEFAULT false"`

	EnableForcePushWhitelist     bool    `xorm:"NOT NULL DEFAULT false"`
	ForcePushWhitelistUserIDs    []int64 `xorm:"JSON TEXT"`
	ForcePushWhitelistDeployKeys bool    `xorm:"NOT NULL DEFAULT false"`

	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

// WhitelistOptions represent all sorts of whitelists used for protected branches
type WhitelistOptions struct {
	UserIDs          []int64
	MergeUserIDs     []int64
	ApprovalsUserIDs []int64
	DeleteUserIDs    []int64
	ForcePushUserIDs []int64
}

type ProtectedBranchRules []*ProtectedBranch

// Match tests if branchName matches the rule
func (protectBranch *ProtectedBranch) Match(branchName string) bool {
	globRule, isPlainName := protectBranch.LoadGlob()
	if isPlainName {
		return strings.EqualFold(protectBranch.RuleName, branchName)
	}
	return globRule.Match(branchName)
}

// Deprecated: use ProtectedBranchManager.GetGlob
func (protectBranch ProtectedBranch) LoadGlob() (glob.Glob, bool) {
	if protectBranch.GlobRule != nil {
		return protectBranch.GlobRule, protectBranch.IsPlainName
	}

	globRule, err := glob.Compile(protectBranch.RuleName, '/')
	if err != nil {
		log.Warn("Invalid glob rule for ProtectedBranch[%d]: %s %v", protectBranch.ID, protectBranch.RuleName, err)
		protectBranch.GlobRule = glob.MustCompile(glob.QuoteMeta(protectBranch.RuleName), '/')
	}

	return globRule, !utils.IsRuleNameSpecial(protectBranch.RuleName)
}
