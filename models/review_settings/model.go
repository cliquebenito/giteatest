package review_settings

import (
	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
	"github.com/gobwas/glob"
)

func init() {
	db.RegisterModel(new(ReviewSettings))
}

type ReviewSettings struct {
	ID                            int64                  `xorm:"pk autoincr"`
	RepoID                        int64                  `xorm:"UNIQUE(s)"`
	Repo                          *repo_model.Repository `xorm:"-"`
	RuleName                      string                 `xorm:"'branch_name' UNIQUE(s)"` // a branch name or a glob match to branch name
	globRule                      glob.Glob              `xorm:"-"`
	EnableMergeWhitelist          bool                   `xorm:"NOT NULL DEFAULT false"`
	MergeWhitelistUserIDs         []int64                `xorm:"JSON TEXT"`
	EnableStatusCheck             bool                   `xorm:"NOT NULL DEFAULT false"`
	StatusCheckContexts           []string               `xorm:"JSON TEXT"`
	EnableDefaultReviewers        bool                   `xorm:"NOT NULL DEFAULT false"`
	BlockOnRejectedReviews        bool                   `xorm:"NOT NULL DEFAULT false"`
	BlockOnOfficialReviewRequests bool                   `xorm:"NOT NULL DEFAULT false"`
	BlockOnOutdatedBranch         bool                   `xorm:"NOT NULL DEFAULT false"`
	DismissStaleApprovals         bool                   `xorm:"NOT NULL DEFAULT false"`
	EnableSonarQube               bool                   `xorm:"NOT NULL DEFAULT false"`

	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func (reviewSettings *ReviewSettings) loadGlob() {
	if reviewSettings.globRule == nil {
		var err error
		reviewSettings.globRule, err = glob.Compile(reviewSettings.RuleName, '/')
		if err != nil {
			log.Warn("Invalid glob rule for reviewSettings[%d]: %s %v", reviewSettings.ID, reviewSettings.RuleName, err)
			reviewSettings.globRule = glob.MustCompile(glob.QuoteMeta(reviewSettings.RuleName), '/')
		}
	}
}

// Match tests if branchName matches the rule
func (reviewSettings *ReviewSettings) Match(branchName string) bool {
	reviewSettings.loadGlob()

	return reviewSettings.globRule.Match(branchName)
}

func (reviewSettings *ReviewSettings) IsUserMergeWhitelisted(userID int64) bool {
	if !reviewSettings.EnableMergeWhitelist {
		// Then we need to fall back on whether the user has write permission
		return true
	}

	if base.Int64sContains(reviewSettings.MergeWhitelistUserIDs, userID) {
		return true
	}

	return false
}
