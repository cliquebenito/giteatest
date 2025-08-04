package pullrequestidresolver

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/repo"
)

type ResolverRequest struct {
	RepoName    string
	BranchName  string
	ProjectName string
}

type PullRequestIDResolver struct {
	engine db.Engine
}

func NewResolver(engine db.Engine) PullRequestIDResolver {
	return PullRequestIDResolver{engine: engine}
}

func (p PullRequestIDResolver) Resolve(_ context.Context, request ResolverRequest) ([]int64, error) {
	repositories := make([]repo.Repository, 0)

	if err := p.engine.
		Where(builder.Eq{"lower_name": request.RepoName, "owner_name": request.ProjectName}).
		Find(&repositories); err != nil {
		return nil, fmt.Errorf("find repository by name: %w", err)
	}

	if len(repositories) != 1 {
		return nil, fmt.Errorf("found %d repositories by name", len(repositories))
	}

	repoID := repositories[0].ID

	pulls := make([]issues.PullRequest, 0)
	if err := p.engine.
		Where("head_repo_id=? AND head_branch=?", repoID, request.BranchName).
		Find(&pulls); err != nil {
		return nil, fmt.Errorf("find pull request by id: %w", err)
	}

	var prIDs []int64
	for _, pull := range pulls {
		prIDs = append(prIDs, pull.ID)
	}

	if len(prIDs) == 0 {
		return nil, fmt.Errorf("no pull requests found for repo: '%s', project: %s",
			request.RepoName, request.ProjectName)
	}

	return prIDs, nil
}
