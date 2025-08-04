package convert

import (
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/gitdiff"
	"context"
	"time"
)

// ToPayloadCommit конвертирует git.Commit в response.PayloadCommit
func ToPayloadCommit(ctx context.Context, c *git.Commit) *response.PayloadCommit {
	authorUsername := ""
	if author, err := userModel.GetUserByEmail(ctx, c.Author.Email); err == nil {
		authorUsername = author.Name
	}
	if authorUsername == "" {
		if author, err := userModel.GetUserByName(ctx, c.Author.Name); err == nil {
			authorUsername = author.Name
		}
	}

	committerUsername := ""
	if committer, err := userModel.GetUserByEmail(ctx, c.Committer.Email); err == nil {
		committerUsername = committer.Name
	}
	if committerUsername == "" {
		if committer, err := userModel.GetUserByName(ctx, c.Committer.Name); err == nil {
			committerUsername = committer.Name
		}
	}

	return &response.PayloadCommit{
		ID:      c.ID.String(),
		Message: c.Message(),
		Author: &response.PayloadUser{
			Name:     c.Author.Name,
			Email:    c.Author.Email,
			UserName: authorUsername,
		},
		Committer: &response.PayloadUser{
			Name:     c.Committer.Name,
			Email:    c.Committer.Email,
			UserName: committerUsername,
		},
		Timestamp:    c.Author.When,
		Verification: ToVerification(ctx, c),
	}
}

type ToCommitOptions struct {
	Stat         bool
	Verification bool
	Files        bool
}

// ToResponseCommit метод конвертирующий git.Commit в response.Commit по аналогии метода convert.ToCommit
func ToResponseCommit(gitRepo *git.Repository, commit *git.Commit, opts ToCommitOptions) (*response.Commit, error) {

	parents := make([]*response.CommitMeta, commit.ParentCount())
	for i := 0; i < commit.ParentCount(); i++ {
		sha, _ := commit.ParentID(i)
		parents[i] = &response.CommitMeta{
			SHA: sha.String(),
		}
	}

	author := &response.CommitUser{
		Identity: response.Identity{
			Name:  commit.Author.Name,
			Email: commit.Author.Email,
		},
		Date: commit.Author.When.Format(time.RFC3339),
	}

	commiter := &response.CommitUser{
		Identity: response.Identity{
			Name:  commit.Committer.Name,
			Email: commit.Committer.Email,
		},
		Date: commit.Committer.When.Format(time.RFC3339),
	}

	res := &response.Commit{
		CommitMeta: &response.CommitMeta{
			SHA:     commit.ID.String(),
			Created: commit.Committer.When,
		},

		RepoCommit: &response.RepoCommit{
			Author:    author,
			Committer: commiter,
			Message:   commit.Message(),
			Tree: &response.CommitMeta{
				SHA:     commit.ID.String(),
				Created: commit.Committer.When,
			},
		},
		Parents: parents,
	}

	// Get diff stats for commit
	if opts.Stat {
		diff, err := gitdiff.GetDiffStat(gitRepo, &gitdiff.DiffOptions{
			AfterCommitID: commit.ID.String(),
		})
		if err != nil {
			return nil, err
		}

		res.Stats = diff
	}

	if opts.Files {
		diff, err := gitdiff.GetDiffFilesWithStat(gitRepo, &gitdiff.DiffOptions{
			AfterCommitID: commit.ID.String(),
		})
		if err != nil {
			return nil, err
		}

		res.Files = diff
	}

	if commit.ParentCount() > 0 {
		parentCommit, _ := commit.Parent(0)
		BeforeCommitId := parentCommit.ID.String()
		res.BeforeCommitId = &BeforeCommitId
	}

	return res, nil
}
