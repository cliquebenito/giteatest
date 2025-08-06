package gitaly

import "gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

// ConflictsClient encapsulates ConflictsService calls
type ConflictsClient struct {
	gitalypb.ConflictsServiceClient
}
