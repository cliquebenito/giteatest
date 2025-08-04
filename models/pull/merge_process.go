package pull

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

type SCMergeProcess struct {
	ID         string             `xorm:"pk autoincr"`
	RepoId     int64              `xorm:"NOT NULL UNIQUE(mp)"`
	UserId     int64              `xorm:"NOT NULL UNIQUE(mp)"`
	PrId       int64              `xorm:"NOT NULL UNIQUE(mp)"`
	BaseBranch string             `xorm:"NOT NULL UNIQUE(mp)"`
	CreatedAt  timeutil.TimeStamp `xorm:"created"`
	UpdatedAt  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(SCMergeProcess))
}

func getMergeInProcess(ctx context.Context, repoId, userId int64, baseBranch string) (*SCMergeProcess, error) {
	mergeProcess := &SCMergeProcess{RepoId: repoId, UserId: userId, BaseBranch: baseBranch}
	exist, err := db.GetEngine(ctx).Get(mergeProcess)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge process: %w", err)
	}
	if !exist {
		return nil, &ErrMergeProcessNotExist{
			RepoId:     repoId,
			UserId:     userId,
			BaseBranch: baseBranch,
		}
	}
	return mergeProcess, nil
}

func GetPullRequestIdForPush(ctx context.Context, repoId, userId int64, baseBranch string) (int64, error) {
	mergeProcess, err := getMergeInProcess(ctx, repoId, userId, baseBranch)
	if err != nil {
		if IsErrMergeProcessNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get pull request id for push: %w", err)
	}
	return mergeProcess.PrId, nil
}

func InsertMergeProcess(ctx context.Context, mergeProcess *SCMergeProcess) error {
	_, err := db.GetEngine(ctx).Insert(mergeProcess)
	if err != nil {
		return fmt.Errorf("failed to insert merge process: %w", err)
	}
	return nil
}

func DeleteMergeProcess(ctx context.Context, mergeProcess *SCMergeProcess) error {
	_, err := db.GetEngine(ctx).Delete(mergeProcess)
	if err != nil {
		return fmt.Errorf("failed to delete merge process: %w", err)
	}
	return nil
}
