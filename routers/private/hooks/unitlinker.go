package hooks

import (
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/pull/pullrequestidresolver"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/unit_linker"
)

func (s Server) linkUnits(ctx *gitea_context.PrivateContext, opts *private.HookOptions) error {
	branchName, err := getBranchName(opts.RefFullNames)
	if err != nil {
		return fmt.Errorf("get branch: %w", err)
	}

	prIDs := []int64{opts.PullRequestID}

	if !opts.IsPullRequest() {
		projectName, repoName, err := getProjectAndRepoNames(ctx.Req.RequestURI)
		if err != nil {
			return fmt.Errorf("get project and repo: %w", err)
		}

		resolverRequest := pullrequestidresolver.ResolverRequest{
			RepoName:    repoName,
			BranchName:  branchName,
			ProjectName: projectName,
		}

		prIDs, err = s.pullRequestIDResolver.Resolve(ctx, resolverRequest)
		if err != nil {
			return fmt.Errorf("resolve pull request ID: %w", err)
		}
	}

	for _, prID := range prIDs {
		request := unit_linker.PullRequestLinkRequest{
			UserName:   opts.UserName,
			BranchName: branchName,

			PullRequestID:     prID,
			PullRequestStatus: pull_request_reader.MergedPullRequestStatus,
		}

		log.Debug("unit_linker: try to link pull request from post-receive hook, request: %v", request)

		if err = s.unitLinker.LinkPullRequest(ctx, request); err != nil {
			return fmt.Errorf("link pull request: %w", err)
		}
	}

	return nil
}

func getBranchName(refs []string) (string, error) {
	for _, ref := range refs {
		branchName, err := git.RefEndNameBranchOnly(ref)
		if err != nil {
			return "", fmt.Errorf("get branch name: %w", err)
		}

		return branchName, nil
	}

	return "", fmt.Errorf("branch was not provided: %v", refs)
}

func getProjectAndRepoNames(uri string) (string, string, error) {
	const prefix = "/api/internal/hook/post-receive/"

	if len(uri) == 0 {
		return "", "", fmt.Errorf("empty uri provided")
	}

	path := strings.TrimPrefix(uri, prefix)

	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid path: %s", uri)
	}

	return parts[0], parts[1], nil
}
