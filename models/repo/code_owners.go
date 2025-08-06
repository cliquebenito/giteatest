package repo

import (
	"context"
	"fmt"
	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
)

func init() {
	db.RegisterModel(new(CodeOwners))
}

// ReviewType defines the sort of feedback a review gives
type ReviewType int

// ReviewTypeUnknown unknown review type
const ReviewTypeUnknown ReviewType = -1

const (
	// ReviewTypePending is a review which is not published yet
	ReviewTypePending ReviewType = iota
	// ReviewTypeApprove approves changes
	ReviewTypeApprove
	// ReviewTypeComment gives general feedback
	ReviewTypeComment
	// ReviewTypeReject gives feedback blocking merge
	ReviewTypeReject
	// ReviewTypeRequest request review from others
	ReviewTypeRequest
)

// Icon returns the corresponding icon for the review type
func (rt ReviewType) Icon() string {
	switch rt {
	case ReviewTypeApprove:
		return "check"
	case ReviewTypeReject:
		return "diff"
	case ReviewTypeComment:
		return "comment"
	case ReviewTypeRequest:
		return "dot-fill"
	default:
		return "comment"
	}
}

type CodeOwners struct {
	ID             int64            `xorm:"PK AUTOINCR"`
	OwnerID        int64            `xorm:"BIGINT NOT NULL"`
	ApprovalStatus ReviewType       `xorm:"INT NOT NULL DEFAULT 4"`
	User           *user_model.User `xorm:"-"`
	AmountUsers    int64            `xorm:"BIGINT NOT NULL DEFAULT 0"`
	PullRequestID  int64            `xorm:"BIGINT NOT NULL"`
	RepoID         int64            `xorm:"BIGINT NOT NULL"`
	IssueID        int64            `xorm:"BIGINT NOT NULL"`
}

func GetCodeOwners(ctx context.Context, repoID, pullRequestID int64) ([]*CodeOwners, error) {
	type Result struct {
		CodeOwners `xorm:"extends"`
		User       user_model.User `xorm:"extends"`
	}

	var results []Result

	err := db.GetEngine(ctx).
		Table("code_owners").
		Join("INNER", `"user"`, "code_owners.owner_id = \"user\".id").
		Where(builder.And(
			builder.Eq{"code_owners.repo_id": repoID},
			builder.Eq{"code_owners.pull_request_id": pullRequestID},
		)).
		Find(&results)

	if err != nil {
		return nil, fmt.Errorf("get code owners: %w", err)
	}

	users := make([]*CodeOwners, 0, len(results))
	for _, result := range results {
		codeOwner := result.CodeOwners
		codeOwner.User = &result.User
		users = append(users, &codeOwner)
	}
	return users, nil
}
