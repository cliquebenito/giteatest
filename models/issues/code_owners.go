package issues

import (
	"context"
	"fmt"
	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

// AddCodeOwnersReviewRequest отвечает за добавление владельцев кода в таблицу code_owners
func AddCodeOwnersReviewRequest(ctx context.Context, issue *Issue, reviewer, doer *user_model.User) error {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		log.Error("Error has occurred while creating transaction. Error: %v", err)
		return fmt.Errorf("init transaction: %w", err)
	}
	defer committer.Close()

	sess := db.GetEngine(ctx)

	exists, err := sess.Where(builder.Eq{
		"owner_id":        reviewer.ID,
		"pull_request_id": issue.PullRequest.IssueID,
		"repo_id":         issue.RepoID,
		"issue_id":        issue.ID,
	}).
		Table(new(repo.CodeOwners)).
		Exist()
	if err != nil {
		log.Error("Error has occurred while checking if code owner exists. Error: %v", err)
		return fmt.Errorf("check if code owner exists: %w", err)
	}

	if !exists {
		codeOwner := &repo.CodeOwners{
			OwnerID:        reviewer.ID,
			PullRequestID:  issue.PullRequest.IssueID,
			RepoID:         issue.RepoID,
			IssueID:        issue.ID,
			ApprovalStatus: repo.ReviewTypeRequest,
		}

		if _, err := sess.Insert(codeOwner); err != nil {
			log.Error("Error has occurred while inserting code owner. Error: %v", err)
			return fmt.Errorf("insert code owner: %w", err)
		}
	}

	return committer.Commit()
}

// UpdateStatusReviewCodeOwners обновляет статус аппрува владельца кода
func UpdateStatusReviewCodeOwners(ctx context.Context, status ReviewType, issue *Issue, doer *user_model.User) error {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		log.Error("Error has occurred while creating transaction. Error: %v", err)
		return fmt.Errorf("initialize transaction: %w", err)
	}
	defer committer.Close()

	sess := db.GetEngine(ctx)
	_, err = sess.Where(builder.And(
		builder.Eq{"owner_id": doer.ID},
		builder.Eq{"issue_id": issue.ID},
	)).
		Cols("approval_status").
		Update(&repo.CodeOwners{ApprovalStatus: repo.ReviewType(status)})
	if err != nil {
		log.Error("Error has occurred while updating code owner approval status. Error: %v", err)
		return fmt.Errorf("update code owner approval status: %w", err)
	}

	return committer.Commit()
}
