package gitaly

import "gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

// CommitClient encapsulates CommitService calls
type CommitClient struct {
	gitalypb.CommitServiceClient
}
